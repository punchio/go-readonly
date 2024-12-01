package parser

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
)

func ParseDir(dir string) (*token.FileSet, []*ast.File, error) {
	var files []*ast.File
	fset := token.NewFileSet()

	// 遍历目录下的所有文件
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 仅解析 .go 文件
		if !info.IsDir() && filepath.Ext(path) == ".go" {
			file, err := parser.ParseFile(fset, path, nil, parser.AllErrors)
			if err != nil {
				return err
			}
			files = append(files, file)
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	return fset, files, nil
}

func ProcessFiles(files []*ast.File) {
	for _, file := range files {
		// 输出文件的包名
		fmt.Printf("Package name: %s\n", file.Name.Name)
		// 可以继续处理文件的其他节点
		ast.Inspect(file, func(n ast.Node) bool {
			if fn, ok := n.(*ast.FuncDecl); ok {
				fmt.Printf("Found function: %s in package %s\n", fn.Name.Name, file.Name.Name)
			}
			return true
		})
	}
}
