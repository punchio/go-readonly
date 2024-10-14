package readonly

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"go/types"
	"log"
	"strings"
	"testing"
)

const roPrefix = "ro"

type StructExample struct {
	roMember int
}

func TestLint(t *testing.T) {
	//src := `
	//package main
	//
	//type MyStruct struct {
	//    roField int
	//}
	//
	//func main() {
	//    var roAbc int
	//    var roDef int
	//    var normalVar int
	//    st := MyStruct{}
	//    nested := StructA{B: StructB{C: StructC{roMember: 0}}}
	//
	//    roAbc = 10            // 不合法
	//    roDef = roAbc        // 合法
	//    st.roField = roAbc   // 合法
	//    nested.B.C.roMember = roAbc // 合法
	//    normalVar = roAbc    // 不合法，应该被检测到
	//}
	//`

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "./example/example1.go", nil, parser.AllErrors)
	if err != nil {
		log.Fatal(err)
	}

	// 用于保存类型信息
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
	}

	// 创建类型检查配置
	conf := types.Config{
		Importer: nil,
	}

	roFuncs := map[*ast.FuncType]int{}

	// 类型检查并填充 info
	_, err = conf.Check("main", fset, []*ast.File{node}, info)
	if err != nil {
		log.Fatal(err)
	}

	ast.Inspect(node, func(node ast.Node) bool {
		// 找到函数或方法声明
		if funcDecl, ok := node.(*ast.FuncDecl); ok {
			// 检查返回值的命名
			if funcDecl.Type.Results != nil {
				index := 0
				for _, field := range funcDecl.Type.Results.List {
					for _, name := range field.Names {
						index++
						// 检查返回值是否以 "ro" 开头
						if strings.HasPrefix(name.Name, roPrefix) {
							roFuncs[funcDecl.Type] |= 1 << index
						}
					}
				}
			}
			// 遍历函数体的所有语句，查找返回语句
			ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
				if returnStmt, ok := n.(*ast.ReturnStmt); ok {
					// 检查返回语句中的所有结果
					for i, result := range returnStmt.Results {
						// 如果返回的是标识符（变量）
						if ident, ok := result.(*ast.Ident); ok {
							// 检查标识符是否以 "ro" 开头
							if strings.HasPrefix(ident.Name, roPrefix) {
								roFuncs[funcDecl.Type] |= 1 << i
							}
						}
					}
				}
				return true
			})
		}
		return true
	})

	ast.Inspect(node, func(n ast.Node) bool {
		switch e := n.(type) {
		case *ast.AssignStmt:
			if e.Tok == token.ASSIGN {
				checkAssign(e, fset, info, roFuncs)
			} else if e.Tok == token.DEFINE {
				checkAssignDeclare(e, fset, roFuncs)
			}
		default:
			_ = e
		}

		if exprStmt, ok := n.(*ast.ExprStmt); ok {
			_ = exprStmt
			ce, ok := exprStmt.X.(*ast.CallExpr)
			if !ok {
				return true
			}
			ft := getExprFuncType(ce)
			if ft == nil {
				return true
			}

			mask, ok := roFuncs[ft]
			if !ok {
				return true
			}

			for i, arg := range ce.Args {
				lhsExprName := ""
				switch e := arg.(type) {
				case *ast.Ident:
					// 直接的标识符
					lhsExprName = e.Name
				case *ast.SelectorExpr:
					// 结构体字段
					lhsExprName = e.Sel.Name
				}

				if strings.HasPrefix(lhsExprName, roPrefix) && mask&1<<i == 0 {
					var buf bytes.Buffer
					_ = printer.Fprint(&buf, fset, arg)
					fmt.Println(buf.String()) // 打印出表达式的源码字符串
					log.Printf("Invalid assignment to %s from %s at %v\n", lhsExprName, buf.String(), fset.Position(arg.Pos()))
				}
			}
		}
		return true
	})
}

func checkAssign(assignStmt *ast.AssignStmt, fset *token.FileSet, info *types.Info, roFuncs map[*ast.FuncType]int) {
	failedIndex := map[int]bool{}
	// 左值不能被赋值
	for i, lhs := range assignStmt.Lhs {
		lhsExprName := ""
		switch e := lhs.(type) {
		case *ast.Ident:
			// 直接的标识符
			lhsExprName = e.Name
		case *ast.SelectorExpr:
			// 结构体字段
			lhsExprName = e.Sel.Name
		}
		if strings.HasPrefix(lhsExprName, roPrefix) {
			failedIndex[i] = true
			var buf bytes.Buffer
			_ = printer.Fprint(&buf, fset, lhs)
			fmt.Println(buf.String()) // 打印出表达式的源码字符串
			log.Printf("Invalid assignment to %s from %s at %v\n", lhsExprName, buf.String(), fset.Position(lhs.Pos()))
			return
		}
	}

	// 右值不能赋值非基础类型
	for i, rhs := range assignStmt.Rhs {
		if failedIndex[i] {
			continue
		}
		rhsExprName := ""
		switch e := rhs.(type) {
		case *ast.Ident:
			// 直接的标识符
			rhsExprName = e.Name
		case *ast.SelectorExpr:
			// 结构体字段
			rhsExprName = e.Sel.Name
		case *ast.CallExpr:
			ft := getExprFuncType(e)
			if mask, ok := roFuncs[ft]; !ok || mask&1<<i == 0 {
				continue
			}
		default:
			continue
		}
		if rhsExprName == "" || strings.HasPrefix(rhsExprName, roPrefix) {
			failedIndex[i] = true

			if !checkExprPrefix(assignStmt.Lhs[i], "_") && !isBasicType(info, assignStmt.Lhs[i]) {
				var buf bytes.Buffer
				_ = printer.Fprint(&buf, fset, rhs)
				fmt.Println(buf.String()) // 打印出表达式的源码字符串
				log.Printf("Invalid assignment to %s from %s at %v\n", rhsExprName, buf.String(), fset.Position(assignStmt.Lhs[i].Pos()))
			}
			return
		}
	}
}

func checkAssignDeclare(assignStmt *ast.AssignStmt, fset *token.FileSet, roFuncs map[*ast.FuncType]int) {
	needCheck := map[int]bool{}
	for i, rhs := range assignStmt.Rhs {
		switch e := rhs.(type) {
		case *ast.CallExpr:
			ft := getExprFuncType(e)
			if mask, ok := roFuncs[ft]; !ok || mask&1<<i == 0 {
				needCheck[i] = true
				continue
			}
		default:
			continue
		}
	}

	for i, lhs := range assignStmt.Lhs {
		if !needCheck[i] {
			continue
		}
		lhsExprName := ""
		switch e := lhs.(type) {
		case *ast.Ident:
			// 直接的标识符
			lhsExprName = e.Name
		}
		if !strings.HasPrefix(lhsExprName, roPrefix) {
			var buf bytes.Buffer
			_ = printer.Fprint(&buf, fset, lhs)
			fmt.Println(buf.String()) // 打印出表达式的源码字符串
			log.Printf("Invalid declare to %s from %s at %v\n", lhsExprName, buf.String(), fset.Position(lhs.Pos()))
			return
		}
	}
}

func getExprFuncType(expr ast.Expr) *ast.FuncType {
	ce, ok := expr.(*ast.CallExpr)
	if !ok {
		return nil
	}
	switch t := ce.Fun.(type) {
	case *ast.Ident:
		d, ok := t.Obj.Decl.(*ast.FuncDecl)
		if !ok {
			return nil
		}
		return d.Type
	case *ast.SelectorExpr:
		return getExprFuncType(t.X)
	default:
		return nil
	}
}

// isValidAssignment 检查赋值是否有效
func checkExprPrefix(expr ast.Expr, prefix string) bool {
	switch e := expr.(type) {
	case *ast.Ident:
		// 直接的标识符
		return strings.HasPrefix(e.Name, prefix)
	case *ast.SelectorExpr:
		// 结构体字段，检查嵌套情况
		return strings.HasPrefix(e.Sel.Name, prefix)
	default:
		return false
	}
}

// 判断左值是否为基础类型
func isBasicType(info *types.Info, expr ast.Expr) bool {
	if obj, ok := info.Types[expr]; ok {
		if basic, ok := obj.Type.Underlying().(*types.Basic); ok {
			switch basic.Kind() {
			case types.Bool,
				types.Int, types.Int8, types.Int16, types.Int32, types.Int64,
				types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64, types.Uintptr,
				types.Float32, types.Float64,
				types.Complex64, types.Complex128,
				types.String:
				return true
			default:
				return false
			}
		}
	}
	return false
}

// getExprText 将表达式转为源码文本
func getExprText(fset *token.FileSet, expr ast.Node) string {
	var buf strings.Builder
	err := printer.Fprint(&buf, fset, expr)
	if err != nil {
		return ""
	}
	return buf.String()
}
