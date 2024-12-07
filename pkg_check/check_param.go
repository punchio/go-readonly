package pkg_check

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"go/types"
	"log"
)

func checkReadonly() error {
	for _, v := range funcTypes {
		for i, stmt := range v.decl.Body.List {
			_ = i
			ast.Inspect(stmt, func(node ast.Node) bool {
				switch e := node.(type) {
				case *ast.ExprStmt:
					roFlag := false
					unwrapExpr(e.X, func(expr ast.Expr) {
						if getExprReadonlyFlag(expr) > 0 {
							roFlag = true
						} else if roFlag {
							ident, ok := expr.(*ast.Ident)
							if !ok {
								return
							}
							_, ok = funcTypes[getNamePos(ident)]
							if !ok {
								return
							}
							var buf bytes.Buffer
							fset := v.pkg.Fset
							_ = printer.Fprint(&buf, fset, expr)
							log.Printf("[lhs:%s,at %v ] cannot be assigned\n", buf.String(), fset.Position(expr.Pos()))
						}
					})
				case *ast.IncDecStmt:
					if getExprReadonlyFlag(e.X) > 0 {
						var buf bytes.Buffer
						_ = printer.Fprint(&buf, v.pkg.Fset, e.X)
						log.Printf("[lhs:%s,at %v ] cannot be assigned\n", buf.String(), v.pkg.Fset.Position(e.X.Pos()))
					}
				case *ast.AssignStmt:
					assign, lhs, rhs := collectAssignStmt(e)
					lhsFlag, skipFlag := collectLhsFlag(lhs, v.pkg.Fset)
					rhsFlag := collectRhsFlag(rhs, skipFlag)
					check(assign, lhs, rhs, lhsFlag, rhsFlag, skipFlag, v.pkg.Fset)
				case *ast.DeclStmt:
					decl := e.Decl.(*ast.GenDecl)
					for _, spec := range decl.Specs {
						valueSpec := spec.(*ast.ValueSpec)
						assign, lhs, rhs := collectValueSpec(valueSpec)
						lhsFlag, skipFlag := collectLhsFlag(lhs, v.pkg.Fset)
						rhsFlag := collectRhsFlag(rhs, skipFlag)
						check(assign, lhs, rhs, lhsFlag, rhsFlag, skipFlag, v.pkg.Fset)
					}
				case *ast.RangeStmt:
					var lhs, rhs []ast.Expr
					lhs = append(lhs, e.Key, e.Value)
					rhs = append(rhs, e.X)
					lhsFlag, skipFlag := collectLhsFlag(lhs, v.pkg.Fset)
					rhsFlag := collectRhsFlag(rhs, skipFlag)
					rhsFlag |= rhsFlag << 1
					check(false, lhs, rhs, lhsFlag, rhsFlag, skipFlag, v.pkg.Fset)
				}
				return true
			})
		}
	}
	return nil
}

func check(isAssign bool, lhs, rhs []ast.Expr, lhsFlag, rhsFlag uint64, skipFlag uint64, fset *token.FileSet) {
	skipIndexes := make([]bool, len(lhs))
	for i := 0; i < len(lhs); i++ {
		if skipFlag&(1<<i) != 0 {
			skipIndexes[i] = true
		}
	}
	if isAssign {
		for i := 0; i < len(lhs); i++ {
			if skipIndexes[i] {
				continue
			}

			if lhsFlag&(1<<i) != 0 {
				skipIndexes[i] = true
				var buf bytes.Buffer
				_ = printer.Fprint(&buf, fset, lhs[i])
				log.Printf("[lhs:%s,at %v ] cannot be assigned\n", buf.String(), fset.Position(lhs[i].Pos()))
			}
		}
	}

	for i := 0; i < len(lhs); i++ {
		if skipIndexes[i] {
			continue
		}

		// 如果右值为非只读，或者左值是只读，则跳过
		if rhsFlag&(1<<i) == 0 || lhsFlag&(1<<i) != 0 {
			continue
		}

		var rhsBuf, lhsBuf bytes.Buffer
		_ = printer.Fprint(&lhsBuf, fset, lhs[i])
		var pos token.Pos
		if len(rhs) == len(lhs) {
			pos = rhs[i].Pos()
			_ = printer.Fprint(&rhsBuf, fset, rhs[i])
		} else {
			pos = rhs[0].Pos()
			_ = printer.Fprint(&rhsBuf, fset, rhs[0])
		}

		fmt.Printf(`variable [lhs:%s at %v ] cannot assigned with 
					variable readonly [rhs:%s at %v ]
`,
			lhsBuf.String(), fset.Position(lhs[i].Pos()),
			rhsBuf.String(), fset.Position(pos))
	}
}

// collectLhsFlag 获取左值的只读标记和跳过标记
// 左值和右值不一样的地方在于，左值类型限制更严格，右值更随意一些
func collectLhsFlag(lhs []ast.Expr, fset *token.FileSet) (roFlag uint64, skipFlag uint64) {
	/*
		左值可能为这些情况
			变量	*ast.Ident	a = 10
			指针解引用	*ast.StarExpr	*p = 20
			数组或切片的元素	*ast.IndexExpr	arr[0] = 5
			结构体字段	*ast.SelectorExpr	person.name = "John"
			字典（映射）元素	*ast.IndexExpr	m["key"] = "value"
			接口变量的底层实现字段	*ast.SelectorExpr	w.Write([]byte("Hello"))
	*/
	for i, expr := range lhs {
		if ident, ok := expr.(*ast.Ident); ok && ident.Name == "_" {
			skipFlag |= 1 << i
			continue
		}

		t := getExprType(expr)
		if _, ok := t.(*types.Basic); ok {
			skipFlag |= 1 << i
			continue
		}

		tmp := getExprReadonlyFlag(expr)
		if tmp > 0 {
			roFlag |= 1 << i
		}
	}
	return
}

func collectRhsFlag(rhs []ast.Expr, skipFlag uint64) uint64 {
	flag := uint64(0)
	for i, expr := range rhs {
		if skipFlag&(1<<i) != 0 {
			continue
		}
		tmp := getExprReadonlyFlag(expr)
		if tmp > 0 {
			flag |= 1 << i
		}
	}
	return flag
}

func collectAssignStmt(stmt *ast.AssignStmt) (isAssign bool, lhs []ast.Expr, rhs []ast.Expr) {
	return stmt.Tok == token.ASSIGN, stmt.Lhs, stmt.Rhs
}

func collectValueSpec(spec *ast.ValueSpec) (isAssign bool, lhs []ast.Expr, rhs []ast.Expr) {
	isAssign = false
	for _, v := range spec.Names {
		lhs = append(lhs, v)
	}
	rhs = spec.Values
	return
}

func unwrapExpr(expr ast.Expr, f func(ast.Expr)) {
	cur := expr
	for cur != nil {
		switch e := cur.(type) {
		case *ast.Ident:
			f(e)
			cur = nil
		case *ast.StarExpr:
			cur = e.X
		case *ast.ParenExpr:
			cur = e.X
		case *ast.UnaryExpr:
			cur = e.X
		case *ast.CallExpr:
			unwrapExpr(e.Fun, f)
			cur = nil
		case *ast.SelectorExpr:
			unwrapExpr(e.X, f)
			f(e.Sel)
			cur = nil
		default:
			cur = nil
		}
		if cur != nil {
			f(cur)
		}
	}
}
