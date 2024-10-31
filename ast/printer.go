package ast

import (
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"strings"
)

func printTree(src string) (*token.FileSet, *ast.File) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "", src, parser.AllErrors)
	if err != nil {
		panic(err)
	}
	_ = ast.Print(fset, node)
	return fset, node
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
