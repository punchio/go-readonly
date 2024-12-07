package pkg_check

import (
	"fmt"
	"golang.org/x/tools/go/packages"
)

func Check(dir string) {
	// 配置解析选项
	cfg := &packages.Config{
		Mode:  packages.NeedName | packages.NeedSyntax | packages.NeedTypesInfo,
		Dir:   dir,   // 指定项目目录
		Tests: false, // 不包含测试文件
	}

	// 加载所有 package
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		fmt.Printf("Error loading packages: %v\n", err)
		return
	}

	for _, pkg := range pkgs {
		loadPackages(pkg)
	}
	fixFuncRoMask()
	err = checkReadonly()
	if err != nil {
		panic(err)
	}
}
