package parser

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"
)

var typeInfos = make(map[*ast.TypeSpec]*typeInfo)
var funcInfos = make(map[*ast.FuncDecl]*funcInfo)

type typeInfo struct {
	spec    *ast.TypeSpec
	methods []*funcInfo
}

func (t *typeInfo) getMember(name string) *ast.Ident {
	structType := t.spec.Type.(*ast.StructType)
	for _, field := range structType.Fields.List {
		for _, ident := range field.Names {
			if ident.Name == name {
				return ident
			}
		}
	}

	for _, method := range t.methods {
		if method.decl.Name.Name == name {
			return method.decl.Name
		}
	}

	return nil
}

type funcInfo struct {
	roMask int64 // bit: 0,1,...31 param ro;32,...63 result ro
	decl   *ast.FuncDecl
}

func (i *funcInfo) calcMask() {
	funcType := i.decl.Type
	if funcType.Params != nil {
		index := 0
		for _, field := range funcType.Params.List {
			for _, name := range field.Names {
				if strings.Contains(name.Name, "ro") {
					i.roMask |= 1 << index
				}
				index++
			}
		}
	}
	if funcType.Results != nil {
		index := 32
		for _, field := range funcType.Results.List {
			for _, name := range field.Names {
				if strings.Contains(name.Name, "ro") {
					i.roMask |= 1 << index
				}
				index++
			}
		}
	}
}

func (i *funcInfo) isResultRo(index int) bool {
	return i.roMask&(1<<(32+index)) != 0
}

func (i *funcInfo) isParamRo(index int) bool {
	return i.roMask&(1<<index) != 0
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
				ident := getStarIdent(t.Recv.List[0].Type)
				addType(ident.Obj.Decl.(*ast.TypeSpec), t)
			}
			addFunc(t)
		}
	}
}

func CheckReadonly(file *ast.File, fset *token.FileSet, info *types.Info) {
	_ = checkReadonly(file, fset, info)
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
