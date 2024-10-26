package parser

import (
	"go/ast"
	"go/importer"
	"go/types"
	"log"
	"readonly/tools"
	"testing"
)

func TestPrint(t *testing.T) {
	tools.PrintTree("./testdata/example.go", nil)
}

func TestCollectTypeInfo(t *testing.T) {
	//fset, f := tools.PrintTree("./testdata/type_spec.go", nil)
	fset, f := tools.PrintTree("./testdata/func.go", nil)
	//fset := token.NewFileSet()
	//f, err := parser.ParseFile(fset, "./testdata/func.go", nil, parser.AllErrors)
	//if err != nil {
	//	panic(err)
	//}
	// 创建类型检查器
	conf := types.Config{Importer: importer.Default()}
	info := &types.Info{
		Defs:       make(map[*ast.Ident]types.Object),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
		Uses:       make(map[*ast.Ident]types.Object),
		Types:      make(map[ast.Expr]types.TypeAndValue),
	}
	_, err := conf.Check("example", fset, []*ast.File{f}, info)
	if err != nil {
		log.Fatalf("types.Check: %v", err)
	}

	CollectTypeSpec(f)
	CheckReadonly(f, fset, info)
}
