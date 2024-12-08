package vet

import (
	"go/ast"
	"go/token"
	"go/types"
	"golang.org/x/tools/go/analysis"
)

// MyAnalyzer 是我们自定义的静态分析器
var MyAnalyzer = &analysis.Analyzer{
	Name: "readonly",                       // 分析器的名字
	Doc:  "check variable with 'ro' start", // 文档说明
	Run:  runAnalyzer,                      // 执行分析的函数
}

var gPass *analysis.Pass
var funcTypes map[token.Pos]*funcInfo // key: FuncDecl.Name.NamePos

// runAnalyzer 执行只读逻辑检查
func runAnalyzer(pass *analysis.Pass) (interface{}, error) {
	gPass = pass
	funcTypes = make(map[token.Pos]*funcInfo)
	initTypes(pass)
	check(pass)
	return nil, nil
}

func initTypes(pass *analysis.Pass) {
	for _, file := range pass.Files {
		for _, decl := range file.Decls {
			if fd, ok := decl.(*ast.FuncDecl); ok {
				object := pass.TypesInfo.Defs[fd.Name].(*types.Func)
				info := &funcInfo{decl: fd, fullName: object.FullName()}
				info.calcMask(true)
				if funcTypes[fd.Name.Pos()] != nil {
					pass.Reportf(fd.Name.Pos(), "%s is defined repeated ", fd.Name.Name)
				}
				funcTypes[fd.Name.Pos()] = info
			}
		}
	}

	fixFuncRoMask(pass)
}
func fixFuncRoMask(pass *analysis.Pass) {
	for i := 0; i < 100; i++ {
		changed := false
		for _, info := range funcTypes {
			old := info.roMask
			info.calcMask(false)
			if old != info.roMask {
				changed = true
			}
		}
		if !changed {
			return
		}
	}
	for pos := range funcTypes {
		pass.Reportf(pos, "func types readonly attribute cannot be stable ")
		break
	}
}
