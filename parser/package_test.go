package parser

import (
	"fmt"
	"golang.org/x/tools/go/packages"
	"testing"
)

const everythingMode = packages.NeedName | packages.NeedFiles |
	packages.NeedCompiledGoFiles | packages.NeedImports | packages.NeedSyntax | packages.NeedDeps | packages.NeedTypes |
	packages.NeedTypesSizes

func TestPackage(t *testing.T) {
	// 配置解析选项
	cfg := &packages.Config{
		//Mode:  packages.NeedSyntax | packages.NeedFiles | packages.NeedTypes | packages.NeedDeps,
		Mode:  packages.LoadMode(65535),
		Dir:   "../parser/testdata", // 指定项目目录
		Tests: false,                // 不包含测试文件
	}

	// 加载所有 package
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		fmt.Printf("Error loading packages: %v\n", err)
		return
	}

	// 遍历加载的 packages
	for _, pkg := range pkgs {
		fmt.Printf("Package: %s\n", pkg.PkgPath)
		for _, file := range pkg.Syntax {
			fmt.Printf("  File: %v\n", file.Pos())
		}
	}
}
