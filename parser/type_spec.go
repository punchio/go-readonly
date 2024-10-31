package parser

import (
	"go/ast"
	"go/token"
	"math"
	"strings"
)

var structInfos = make(map[*ast.StructType]*typeInfo)
var funcTypeInfos = make(map[*ast.FuncType]*funcInfo)

var typeInfos = make(map[*ast.TypeSpec]*typeInfo)
var funcInfos = make(map[*ast.FuncDecl]*funcInfo)

type typeInfo struct {
	spec    *ast.TypeSpec
	methods []*funcInfo
}

func (t *typeInfo) getMember(name string) (member *ast.Ident, method *funcInfo) {
	structType := t.spec.Type.(*ast.StructType)
	for _, field := range structType.Fields.List {
		for _, ident := range field.Names {
			if ident.Name == name {
				return ident, nil
			}
		}
	}

	for _, m := range t.methods {
		if m.decl.Name.Name == name {
			return nil, m
		}
	}

	return nil, nil
}

type funcInfo struct {
	roMask        uint64 // bit: 0,1,...31 param ro;32,...63 result ro
	decl          *ast.FuncDecl
	resultChecked bool
}

func (i *funcInfo) calcMask() {
	i.roMask = calcFuncResultMask(i.decl.Type, nil)
}

func (i *funcInfo) checkResult() {
	i.roMask = calcFuncResultMask(i.decl.Type, i.decl.Body)
	i.resultChecked = true
}

func calcFuncResultMask(funcType *ast.FuncType, body *ast.BlockStmt) uint64 {
	roMask := uint64(0)
	if funcType.Params != nil {
		index := 0
		for _, field := range funcType.Params.List {
			for _, name := range field.Names {
				if strings.Contains(name.Name, roPrefix) {
					roMask |= 1 << index
				}
				index++
			}
		}
	}

	// 检测命名返回值中的只读
	if funcType.Results != nil {
		index := 32
		for _, field := range funcType.Results.List {
			for _, name := range field.Names {
				if strings.Contains(name.Name, roPrefix) {
					roMask |= 1 << index
				}
				index++
			}
		}
	}

	if body != nil && funcType.Results != nil {
		// 检测返回语句中的只读变量
		index := 32
		for _, stmt := range body.List {
			returnStmt, ok := stmt.(*ast.ReturnStmt)
			if !ok {
				continue
			}

			if len(returnStmt.Results) > 1 || len(funcType.Results.List) == 1 {
				for _, expr := range returnStmt.Results {
					if ident, ok := expr.(*ast.Ident); ok {
						if strings.Contains(ident.Name, roPrefix) {
							roMask |= 1 << index
						}
					} else if call, ok := expr.(*ast.CallExpr); ok {
						if getRoFuncResultFlag(call) > 0 {
							roMask |= 1 << index
						}
					} else if sel, ok := expr.(*ast.SelectorExpr); ok {
						if getSelectorRoFlag(sel) > 0 {
							roMask |= 1 << index
						}
					}
					index++
				}
			} else {
				call := returnStmt.Results[0].(*ast.CallExpr)
				roMask |= getRoFuncResultFlag(call) << index
			}
		}
	}
	return roMask
}

func (i *funcInfo) hasRoResult() bool {
	return i.roMask&(uint64(math.MaxUint32)<<32) != 0
}

func (i *funcInfo) hasRoParam() bool {
	return i.roMask&math.MaxUint32 > 0
}

func (i *funcInfo) isRoResult(index int) bool {
	return i.roMask&(uint64(1)<<(32+index)) != 0
}

func (i *funcInfo) isRoParam(index int) bool {
	return i.roMask&(uint64(1)<<index) != 0
}

func (i *funcInfo) getRoResultFlag() uint64 {
	if !i.resultChecked {
		i.checkResult()
	}
	return (i.roMask & (uint64(math.MaxUint32) << 32)) >> 32
}

func (i *funcInfo) getRoParamFlag() uint64 {
	if !i.resultChecked {
		i.checkResult()
	}
	return i.roMask & math.MaxUint32
}

func (i *funcInfo) getIdent() *ast.Ident {
	ident := ast.NewIdent(i.decl.Name.Name)
	ident.Obj = ast.NewObj(ast.Fun, ident.Name)
	ident.Obj.Decl = i.decl
	return ident
}
func (i *funcInfo) getDecl() *ast.FuncDecl {
	return i.decl
}

func addType(st *ast.TypeSpec, m ...*ast.FuncDecl) {
	i, ok := typeInfos[st]
	if !ok {
		i = &typeInfo{spec: st}
		typeInfos[st] = i
	}

	for _, decl := range m {
		info := buildFuncInfo(decl)
		i.methods = append(i.methods, info)
	}

	structType, ok := st.Type.(*ast.StructType)
	if ok {
		v, ok := structInfos[structType]
		if !ok {
			v = &typeInfo{spec: st}
			structInfos[structType] = v
		}
		for _, decl := range m {
			info := buildFuncInfo(decl)
			v.methods = append(v.methods, info)
		}
	}
}

func addFunc(decl *ast.FuncDecl) {
	info := buildFuncInfo(decl)
	funcInfos[decl] = info

	funcTypeInfos[decl.Type] = info
}

func buildFuncInfo(funcDecl *ast.FuncDecl) *funcInfo {
	fi := &funcInfo{decl: funcDecl}
	fi.calcMask()
	return fi
}
func CollectTypeSpec(pkgs map[string]*ast.Package) {
	for _, pkg := range pkgs {
		for _, f := range pkg.Files {
			collectTypeSpec(f)
		}
	}
	fixFuncRoMask()
}
func collectTypeSpec(file *ast.File) {
	for _, decl := range file.Decls {
		switch t := decl.(type) {
		case *ast.GenDecl:
			if t.Tok != token.TYPE {
				continue
			}
			addType(t.Specs[0].(*ast.TypeSpec))
		case *ast.FuncDecl:
			if t.Recv != nil {
				ident := getStarIdent(t.Recv.List[0].Type)
				addType(ident.Obj.Decl.(*ast.TypeSpec), t)
			}
			addFunc(t)
		}
	}
}

func fixFuncRoMask() {
	changed := true
	maxCount := 1000
	for changed {
		maxCount--
		if maxCount <= 0 {
			panic("func ro mask not stable")
		}
		changed = false
		for _, info := range funcInfos {
			old := info.roMask
			info.checkResult()
			if old != info.roMask {
				changed = true
			}
		}
	}
}

func CheckReadonly(fset *token.FileSet) {
	_ = checkReadonly(fset)
}

func getStarIdent(expr ast.Expr) *ast.Ident {
	switch e := expr.(type) {
	case *ast.Ident:
		return e
	case *ast.StarExpr:
		return getStarIdent(e.X)
	default:
		return nil
	}
}
