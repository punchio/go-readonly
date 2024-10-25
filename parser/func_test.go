package parser

import (
	"fmt"
	"go/ast"
	"go/types"
	"readonly/tools"
	"testing"
)

func Test_CollectFuncDecl(t *testing.T) {
	fset, f := tools.PrintTree("./testdata/func.go", nil)
	// 创建类型信息存储
	conf := types.Config{Importer: nil}
	info := &types.Info{
		Selections: map[*ast.SelectorExpr]*types.Selection{},
		Scopes:     make(map[ast.Node]*types.Scope), // 用于存储每个节点的作用域
	}

	// 类型检查
	_, err := conf.Check("main", fset, []*ast.File{f}, info)
	if err != nil {
		fmt.Println(err)
		return
	}
	//ast.Inspect(f, func(node ast.Node) bool {
	//	collectTypeSpec(node)
	//	return true
	//})
	//ast.Inspect(f, func(node ast.Node) bool {
	//	CollectFuncDecl(node)
	//	return true
	//})

	fmt.Println(funcInfos)
}
