package ast

import (
	"go/ast"
	"go/types"
	"testing"
)

func TestValueSpec(t *testing.T) {
	src := `
package main

func (st) get() int { return 0 }

type st struct {
	t st2
}

type st2 struct {
	v int
}


func (st) getSt() st2 { return st2{} }

func (st2) get() int { return 0 }

func r1() int {
	return 0
}

func r2() (int, int) {
	return 0, 0
}
func r3() (int, int, int) {
	return 0, 0, 0
}

func f2() {
	var drain []any
	var vst0 *st
	var vi1, vi2 int

	var v1, v2, v3 = r1(), r1(), r1()
	var v4, v5, v6 = r3()
	var v7, v8, v9 = 1, v5, r1()
	var a = st{}
	var v10 = a.get()
	var v11 = a.t.get()
	var v12 = a.getSt().get()
	var v13 = a.t.v
	v100, v101 := 1, int32(0)

	drain = append(drain, vst0, vi1, vi2,
		v1, v2, v3, v4, v5, v6, v7, v8, v9, v10, v11, v12, v13,
		v100, v101)
}
`
	fset, node := printTree(src)
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
	}

	// 创建类型检查配置
	conf := types.Config{
		Importer: nil,
	}

	// 类型检查并填充 info
	_, err := conf.Check("main", fset, []*ast.File{node}, info)
	if err != nil {
		t.Fatal(err)
	}
	ast.Inspect(node, func(node ast.Node) bool {
		switch expr := node.(type) {
		case *ast.ValueSpec:
			if expr.Names[0].Name == "v13" {
				ast.Inspect(expr, func(node ast.Node) bool {
					switch expr := node.(type) {
					case *ast.Ident:
						t.Log("----select:", expr.Name, "=>")
						if expr.Obj != nil && expr.Obj.Decl != nil {
							switch expr.Obj.Decl.(type) {
							case *ast.ValueSpec:
								t.Log("type:", expr.Obj.Type)
							default:
								t.Log("other:", getExprText(fset, node))
							}
						}
					default:
						t.Log("other:", getExprText(fset, node))
					}
					return true
				})
			}
			//case *ast.CallExpr:
			//	t.Log("call expr:", getExprText(fset, expr), "=>")
			//	value, ok := info.Types[expr]
			//	if ok {
			//		t.Log("type:", value.Type)
			//	} else {
			//		t.Log("not found")
			//	}
			//
			//	log.Print("call expr func:", getExprText(fset, expr), "=>")
			//	value, ok = info.Types[expr.Fun]
			//	if ok {
			//		t.Log("type:", value.Type)
			//	} else {
			//		t.Log("not found")
			//	}
		}
		return true
	})
}

func TestTypeSpec(t *testing.T) {
	src := `
package main
type s1 struct {}
type s2 = s1
type s3 = int
`
	printTree(src)
}
func TestImportSpec(t *testing.T) {
	src := `
package main
import "testing"

func init() {
	var t *testing.T
	_ = t
}
`
	printTree(src)
}
