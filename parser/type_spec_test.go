package parser

import (
	"fmt"
	"go/ast"
	"readonly/tools"
	"testing"
)

func TestPrint(t *testing.T) {
	tools.PrintTree("./testdata/example.go", nil)
}

func TestCollectStruct(t *testing.T) {
	_, f := tools.PrintTree("./testdata/type_spec.go", nil)
	ast.Inspect(f, func(node ast.Node) bool {
		CollectTypeSpec(node)
		return true
	})

	fmt.Println(typeInfos)
}
