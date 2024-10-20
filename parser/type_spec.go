package parser

import (
	"go/ast"
)

var typeInfos = make(map[*ast.TypeSpec]*typeInfo)

type typeInfo struct {
	spec    *ast.TypeSpec
	methods []*ast.FuncDecl
}

func addType(st *ast.TypeSpec, m ...*ast.FuncDecl) {
	i, ok := typeInfos[st]
	if !ok {
		i = &typeInfo{spec: st}
		typeInfos[st] = i
	}
	i.methods = append(i.methods, m...)
}

// CollectTypeSpec 获取结构体方法
// FuncDecl.Recv.List[0].Type->*ast.Ident->Ident.Obj.Decl->*ast.TypeSpec->TypeSpec.Type->*ast.StructType
func CollectTypeSpec(node ast.Node) {
	switch e := node.(type) {
	case *ast.TypeSpec:
		addType(e)
	case *ast.InterfaceType:
		//todo
	case *ast.FuncDecl:
		if e.Recv != nil {
			// 获取接收类型
			var ident *ast.Ident
			star, ok := e.Recv.List[0].Type.(*ast.StarExpr)
			if ok {
				ident, ok = star.X.(*ast.Ident)
			} else {
				ident, ok = e.Recv.List[0].Type.(*ast.Ident)
			}
			if !ok {
				return
			}

			// 获取类型声明
			ts, ok := ident.Obj.Decl.(*ast.TypeSpec)
			if !ok {
				return
			}
			addType(ts, e)
		}
	}
}
