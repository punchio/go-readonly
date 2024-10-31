package parser

import (
	"go/parser"
	"go/token"
	"readonly/tools"
	"testing"
)

func TestPrint(t *testing.T) {
	tools.PrintTree("./testdata/example.go", nil)
}

func TestCollectTypeInfo(t *testing.T) {
	//fset, f := tools.PrintTree("./testdata/type_spec.go", nil)
	//fset, f := tools.PrintTree("./testdata/func.go", nil)
	//fset := token.NewFileSet()
	//f, err := parser.ParseFile(fset, "./testdata/func.go", nil, parser.AllErrors)
	//if err != nil {
	//	panic(err)
	//}
	// 创建类型检查器

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, "./testdata", nil, parser.AllErrors)
	if err != nil {
		panic(err)
	}

	CollectTypeSpec(pkgs)
	CheckReadonly(fset)
}
