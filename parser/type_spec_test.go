package parser

import (
	"go/token"
	"go/types"
	"readonly/tools"
	"testing"
)

func TestPrint(t *testing.T) {
	tools.PrintTree("./testdata/example.go", nil)
}

func TestFile(t *testing.T) {
	fset, f := tools.PrintTree("./testdata/example.go", nil)
	collectTypeSpec(f)
	fixFuncRoMask()
	CheckReadonly(fset)
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

	//pkgs, err := parser.ParseDir(fset, "./testdata", nil, parser.AllErrors)
	//if err != nil {
	//	panic(err)
	//}
	//
	//files := make([]*ast.File, 0)
	//for _, pkg := range pkgs {
	//	for _, file := range pkg.Files {
	//		files = append(files, file)
	//	}
	//}

	fset, files, err := ParseDir("./testdata")
	if err != nil {
		panic(err)
	}

	conf := types.Config{Error: func(err error) {
		return
	}}

	pkg, err := conf.Check("./testdata", fset, files, info)
	_ = pkg

	CollectTypeSpec(files)
	CheckReadonly(fset)
}
