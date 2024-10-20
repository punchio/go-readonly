package parser

import (
	"fmt"
	"go/ast"
	"readonly/tools"
	"testing"
)

func Test_CollectFuncDecl(t *testing.T) {
	_, f := tools.PrintTree("./testdata/func.go", nil)
	ast.Inspect(f, func(node ast.Node) bool {
		CollectFuncDecl(node)
		return true
	})

	fmt.Println(funcInfos)
}
