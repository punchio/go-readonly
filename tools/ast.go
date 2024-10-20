package tools

import (
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"strings"
)

// PrintTree 打印语法树
func PrintTree(filename string, src any) (*token.FileSet, *ast.File) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, src, parser.AllErrors)
	if err != nil {
		panic(err)
	}
	_ = ast.Print(fset, f)
	return fset, f
}

// GetExprText 将表达式转为源码文本
func GetExprText(fset *token.FileSet, expr ast.Node) string {
	var buf strings.Builder
	err := printer.Fprint(&buf, fset, expr)
	if err != nil {
		return ""
	}
	return buf.String()
}
