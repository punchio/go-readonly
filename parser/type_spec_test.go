package parser

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"readonly/tools"
	"testing"
	"time"
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

	pkgs, err := parser.ParseDir(fset, "./testdata", nil, parser.AllErrors)
	if err != nil {
		panic(err)
	}

	files := make([]*ast.File, 0)
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			files = append(files, file)
		}
	}

	conf := types.Config{Error: func(err error) {
		return
	}}
	info := &types.Info{
		Types:        make(map[ast.Expr]types.TypeAndValue),
		Instances:    make(map[*ast.Ident]types.Instance),
		Defs:         make(map[*ast.Ident]types.Object),
		Uses:         make(map[*ast.Ident]types.Object),
		Implicits:    map[ast.Node]types.Object{},
		Scopes:       map[ast.Node]*types.Scope{},
		Selections:   map[*ast.SelectorExpr]*types.Selection{},
		InitOrder:    []*types.Initializer{},
		FileVersions: map[*ast.File]string{},
	}
	pkg, err := conf.Check("./testdata", fset, files, info)
	_ = pkg

	CollectTypeSpec(pkgs)
	CheckReadonly(fset)
}

func TestTimer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	count := 100000
	for i := 0; i < count; i++ {
		createTimer(ctx)
	}
	time.Sleep(time.Minute)
}

func createTimer(ctx context.Context) {
	go func() {
		t := time.NewTicker(time.Second)
		for {
			select {
			case <-t.C:
			case <-ctx.Done():
				return
			}
		}
	}()
}
