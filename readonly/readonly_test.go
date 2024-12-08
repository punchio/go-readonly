package readonly

import (
	"fmt"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/packages"
	"os"
	"path/filepath"
	"testing"
)

func TestFromTestdata(t *testing.T) {
	// 配置解析选项
	cfg := &packages.Config{
		Mode:  packages.NeedName | packages.NeedSyntax | packages.NeedTypesInfo,
		Dir:   "./testdata", // 指定项目目录
		Tests: false,        // 不包含测试文件
	}

	// 加载所有 package
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		fmt.Printf("Error loading packages: %v\n", err)
		return
	}
	initTypes(pkgs)

	analyzer := &analysis.Analyzer{
		Name: linterName,
		Doc:  "check variable with 'ro' start", // 文档说明
		Run:  runAnalyzer,                      // 执行分析的函数
	}
	curPath, _ := os.Getwd()
	// 遍历加载的包并执行分析
	for _, pkg := range pkgs {
		var diag []analysis.Diagnostic
		pass := &analysis.Pass{
			Analyzer:  analyzer, // 需要定义你的分析器
			Fset:      pkg.Fset,
			Files:     pkg.Syntax,
			TypesInfo: pkg.TypesInfo,
			Pkg:       pkg.Types,
			Report: func(diagnostic analysis.Diagnostic) {
				diag = append(diag, diagnostic)
			}, // 需要定义一个报告函数
			TypesSizes: pkg.TypesSizes,
			// 你可以根据需要设置其他字段
		}

		_, _ = analyzer.Run(pass)

		for _, v := range diag {
			pos := pkg.Fset.Position(v.Pos)
			fp, _ := filepath.Rel(curPath, pos.Filename)
			pos.Filename = fp
			fmt.Printf("lhs:[ %v ] \n %s\n", pos, v.Message)
		}
	}
}
