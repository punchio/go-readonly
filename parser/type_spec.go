package parser

import (
	"go/ast"
	"go/token"
	"go/types"
	"math"
	"strings"
)

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
	roMask uint64 // bit: 0,1,...31 param ro;32,...63 result ro
	decl   *ast.FuncDecl
}

func (i *funcInfo) calcMask() {
	funcType := i.decl.Type
	if funcType.Params != nil {
		index := 0
		for _, field := range funcType.Params.List {
			for _, name := range field.Names {
				if strings.Contains(name.Name, roPrefix) {
					i.roMask |= 1 << index
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
					i.roMask |= 1 << index
				}
				index++
			}
		}
	}

	// 检测返回语句中的只读变量
	index := 32
	for _, stmt := range i.decl.Body.List {
		returnStmt, ok := stmt.(*ast.ReturnStmt)
		if !ok {
			continue
		}
		for _, expr := range returnStmt.Results {
			if ident, ok := expr.(*ast.Ident); ok {
				if strings.Contains(ident.Name, roPrefix) {
					i.roMask |= 1 << index
				}
			}
			index++
		}
	}
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
	return i.roMask & (uint64(math.MaxUint32) << 32)
}

func (i *funcInfo) getRoParamFlag() uint64 {
	return i.roMask & math.MaxUint32
}

func (i *funcInfo) getIdent() *ast.Ident {
	ident := ast.NewIdent(i.decl.Name.Name)
	ident.Obj = ast.NewObj(ast.Fun, ident.Name)
	ident.Obj.Decl = i.decl
	return ident
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
}

func addFunc(decl *ast.FuncDecl) {
	info := buildFuncInfo(decl)
	funcInfos[decl] = info
}

func buildFuncInfo(funcDecl *ast.FuncDecl) *funcInfo {
	fi := &funcInfo{decl: funcDecl}
	fi.calcMask()
	return fi
}
func CollectTypeSpec(file *ast.File) {
	collectTypeSpec(file)
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
				ident := getFinalIdent(t.Recv.List[0].Type)
				addType(ident.Obj.Decl.(*ast.TypeSpec), t)
			}
			addFunc(t)
		}
	}
}

func CheckReadonly(file *ast.File, fset *token.FileSet, info *types.Info) {
	_ = checkReadonly(file, fset, info)
}

func getFinalIdent(expr ast.Expr) *ast.Ident {
	switch e := expr.(type) {
	case *ast.Ident:
		return e
	case *ast.StarExpr:
		return getFinalIdent(e.X)
	default:
		return nil
	}
}
