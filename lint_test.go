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

type StructExample struct {
	roMember int
}

func TestLint(t *testing.T) {
	src := `
    package main

    type MyStruct struct {
        roField int
    }

    func main() {
        var roAbc int
        var roDef int
        var normalVar int
        st := MyStruct{}
        nested := StructA{B: StructB{C: StructC{roMember: 0}}}

        roAbc = 10            // 不合法
        roDef = roAbc        // 合法
        st.roField = roAbc   // 合法
        nested.B.C.roMember = roAbc // 合法
        normalVar = roAbc    // 不合法，应该被检测到
    }
    `

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "", src, parser.AllErrors)
	if err != nil {
		log.Fatal(err)
	}

	// 用于保存类型信息
	info := types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
	}

	// 创建类型检查配置
	conf := types.Config{
		Importer: nil,
	}

	// 类型检查并填充 info
	_, err = conf.Check("main", fset, []*ast.File{node}, &info)
	if err != nil {
		log.Fatal(err)
	}

	ast.Inspect(node, func(n ast.Node) bool {
		if assignStmt, ok := n.(*ast.AssignStmt); ok && assignStmt.Tok == token.ASSIGN {
			for _, lhs := range assignStmt.Lhs {
				if ident, ok := lhs.(*ast.Ident); ok && strings.HasPrefix(ident.Name, "ro") {
					// 检查赋值目标
					for _, rhs := range assignStmt.Rhs {
						if isValidAssignment(rhs) {
							continue
						}
						var buf bytes.Buffer
						_ = printer.Fprint(&buf, fset, rhs)
						fmt.Println(buf.String()) // 打印出表达式的源码字符串
						log.Printf("Invalid assignment to %s from %s at %v\n", ident.Name, buf.String(), fset.Position(rhs.Pos()))
					}
				}
			}

			for _, rhs := range assignStmt.Rhs {
				if ident, ok := rhs.(*ast.Ident); ok && strings.HasPrefix(ident.Name, "ro") {

				}
			}
		}
		return true
	})
}

// isValidAssignment 检查赋值是否有效
func isValidAssignment(expr ast.Expr) bool {
	switch e := expr.(type) {
	case *ast.Ident:
		// 直接的标识符
		return strings.HasPrefix(e.Name, "ro")
	case *ast.SelectorExpr:
		// 结构体字段，检查嵌套情况
		return isFieldOfRoPrefix(e)
	default:
		return false
	}
}

// isFieldOfRoPrefix 检查结构体字段是否以 ro 开头
func isFieldOfRoPrefix(selExpr *ast.SelectorExpr) bool {
	if _, ok := selExpr.X.(*ast.Ident); ok {
		// 递归检查嵌套的结构体字段
		if strings.HasPrefix(selExpr.Sel.Name, "ro") {
			return true
		}
		// 如果是嵌套的结构体，继续检查
		return isFieldOfRoPrefix(selExpr.X.(*ast.SelectorExpr))
	}
	return false
}
