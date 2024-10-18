package readonly

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"go/types"
	"log"
	"strings"
	"testing"
)

type StructExample struct {
	roMember int
}

//func BenchmarkLint(b *testing.B) {
//	for i := 0; i < b.N; i++ {
//		doLint()
//	}
//}

func TestLint(t *testing.T) {
	doLint()
}

func doLint() {
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
	//node, err := parser.ParseDir(fset, "./example", nil, parser.AllErrors)
	//node, err := parser.ParseFile(fset, "./example/example1.go", nil, parser.AllErrors)
	node, err := parser.ParseFile(fset, "./example/base_rule.go", nil, parser.AllErrors)
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

	roFuncSet := map[*ast.FuncType]int{}

	// 类型检查并填充 info
	_, err = conf.Check("main", fset, []*ast.File{node}, info)
	if err != nil {
		log.Fatal(err)
	}

	ast.Inspect(node, func(node ast.Node) bool {
		switch expr := node.(type) {
		case *ast.FuncDecl:
			collectRoFuncSet(expr, roFuncSet)
		}
		return true
	})

	ast.Inspect(node, func(n ast.Node) bool {
		switch e := n.(type) {
		case *ast.ValueSpec:
			for _, value := range e.Values {
				_ = value
				//rhsExprName := ""
				//switch e := value.(type) {
				//case *ast.Ident:
				//	// 直接的标识符
				//	rhsExprName = e.Name
				//case *ast.SelectorExpr:
				//	// 结构体字段
				//	rhsExprName = e.Sel.Name
				//case *ast.CallExpr:
				//	ft := getExprFuncType(e, info)
				//	if mask, ok := roFuncSet[ft]; !ok || mask&1<<i == 0 {
				//		continue
				//	}
				//default:
				//	continue
				//}
				//if rhsExprName == "" || strings.HasPrefix(rhsExprName, roPrefix) {
				//	failedIndex[i] = true
				//
				//	if !checkExprPrefix(assignStmt.Lhs[i], "_") && !isBasicType(info, assignStmt.Lhs[i]) {
				//		var buf bytes.Buffer
				//		_ = printer.Fprint(&buf, fset, rhs)
				//		log.Printf("Invalid assignment to %s from %s at %v\n", rhsExprName, buf.String(), fset.Position(assignStmt.Lhs[i].Pos()))
				//	}
				//	return
				//}
			}

		case *ast.AssignStmt:
			if e.Tok == token.ASSIGN {
				checkAssign(e, fset, info, roFuncSet)
			} else if e.Tok == token.DEFINE {
				checkAssignDeclare(e, fset, info, roFuncSet)
			}
		case *ast.CallExpr:
			ft := getExprFuncType(e, info)
			mask := 0
			if ft == nil {
				mask = ^0
			} else {
				m, ok := roFuncSet[ft]
				if !ok {
					return true
				}
				mask = m
			}

			for i, arg := range e.Args {
				lhsExprName := ""
				switch e := arg.(type) {
				case *ast.Ident:
					// 直接的标识符
					lhsExprName = e.Name
				case *ast.SelectorExpr:
					// 结构体字段
					lhsExprName = e.Sel.Name
				}

				if strings.HasPrefix(lhsExprName, roPrefix) && mask&(1<<(i+32)) == 0 {
					var buf bytes.Buffer
					_ = printer.Fprint(&buf, fset, arg)
					log.Printf("Invalid func call to %s from %s at %v\n", lhsExprName, buf.String(), fset.Position(arg.Pos()))
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
			ft := getExprFuncType(e, info)
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
				log.Printf("Invalid assignment to %s from %s at %v\n", rhsExprName, buf.String(), fset.Position(assignStmt.Lhs[i].Pos()))
			}
			return
		}
	}
}

func checkAssignDeclare(assignStmt *ast.AssignStmt, fset *token.FileSet, info *types.Info, roFuncs map[*ast.FuncType]int) {
	needCheck := map[int]bool{}
	for i, rhs := range assignStmt.Rhs {
		switch e := rhs.(type) {
		case *ast.CallExpr:
			ft := getExprFuncType(e, info)
			if mask, ok := roFuncs[ft]; !ok || mask&1<<i == 0 {
				needCheck[i] = true
			}
		case *ast.Ident:
			if strings.HasPrefix(e.Name, roPrefix) && !isBasicType(info, e) {
				needCheck[i] = true
			}
		case *ast.SelectorExpr:
			if strings.HasPrefix(e.Sel.Name, roPrefix) {
				needCheck[i] = true
			}
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
			log.Printf("Invalid declare to %s from %s at %v\n", lhsExprName, buf.String(), fset.Position(lhs.Pos()))
			return
		}
	}
}
