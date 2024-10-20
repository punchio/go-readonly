package parser

import (
	"go/ast"
	"strings"
)

var funcInfos = make(map[*ast.FuncDecl]*funcInfo)

type funcInfo struct {
	roMask int64 // bit: 0,1,...31 param ro;32,...63 result ro
	decl   *ast.FuncDecl
	caller []*ast.FuncDecl
	callee []*ast.FuncDecl
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

func addCallee(caller, callee *ast.FuncDecl) {
	info, ok := funcInfos[caller]
	if !ok {
		info = &funcInfo{
			roMask: 0,
			decl:   caller,
		}
		info.calcMask()
		funcInfos[caller] = info
	}
	info.callee = append(info.callee, callee)
}

// CollectFuncDecl 获取结构体方法
func CollectFuncDecl(node ast.Node) {
	switch d := node.(type) {
	case *ast.FuncDecl:
		ast.Inspect(d.Body, func(node ast.Node) bool {
			expr, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			ident, ok := expr.Fun.(*ast.Ident)
			if !ok {
				return true
			}
			f, ok := ident.Obj.Decl.(*ast.FuncDecl)
			if !ok {
				return true
			}
			addCallee(d, f)
			return true
		})
	}
}
