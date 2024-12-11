package readonly

import (
	"fmt"
	"go/ast"
	"go/token"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/packages"
	"slices"
	"sync"
)

func Setup(pkgs []*packages.Package) {
	initTypes(pkgs)
}

func NewAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: linterName,
		Doc:  docEN,
		Run:  runAnalyzer,
	}
}

func Run(pkgPaths ...string) []string {
	var messages []string
	for _, v := range pkgPaths {
		messages = append(messages, checkDir(v)...)
	}
	return messages
}

const linterName = "readonly"
const defaultPrefix = "ro"

const docEN = `
A read-only property refers to any variable whose name starts with ro and cannot be modified, including the fields within the variable. 
If the variable is passed as a parameter to another function, it retains its read-only property. 
Similarly, when returned as a value, the receiving variable must also comply with the read-only rule.
Primitive types are not subject to the read-only restriction, as they are passed by value (copied). 
For example, slices, maps, structs, and other reference types are protected by the read-only rule.
`
const docCN = `
只读属性是指所有以ro开头的变量，不能被修改，包括变量内的字段。
如果变量作为参数，传入另一个函数，仍然继承只读属性；同样，作为返回值，接收它的变量仍然需要满足只读规则。
基础类型不受只读限制，因为是复制方式。
例如，切片、映射、结构体等都受只读保护。
`

var allPackages []*packages.Package
var funcTypes map[token.Pos]*funcInfo // key: FuncDecl.Name.NamePos
var moduleDir string
var roPrefix = defaultPrefix

func runAnalyzer(pass *analysis.Pass) (interface{}, error) {
	inspector := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	inspector.Preorder([]ast.Node{(*ast.FuncDecl)(nil)}, func(n ast.Node) {
		funcDecl := n.(*ast.FuncDecl)
		for _, stmt := range funcDecl.Body.List {
			checkStmt(pass, stmt)
		}
	})
	return nil, nil
}

func checkDir(root string) []string {
	// 配置解析选项
	cfg := &packages.Config{
		Mode:  packages.NeedName | packages.NeedSyntax | packages.NeedTypesInfo | packages.NeedModule,
		Dir:   root,  // 指定项目目录
		Tests: false, // 不包含测试文件
	}

	// 加载所有 package
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		panic(fmt.Sprintf("loading packages fail, err: %v", err))
	}
	inspectors := initTypes(pkgs)

	analyzer := &analysis.Analyzer{
		Name: linterName,
		Doc:  "check variable with 'ro' start", // 文档说明
		Run:  runAnalyzer,                      // 执行分析的函数
	}

	// 遍历加载的包并执行分析
	var messages []string
	var mu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(len(pkgs))
	for i, pkg := range pkgs {
		go func() {
			defer func() {
				wg.Done()
				if err := recover(); err != nil {
					mu.Lock()
					messages = append(messages, fmt.Sprintf("%s: pkg check fail, panic: %v", pkg.PkgPath, err))
					mu.Unlock()
				}
			}()
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
				ResultOf:   map[*analysis.Analyzer]interface{}{inspect.Analyzer: inspectors[i]},
				// 你可以根据需要设置其他字段
			}

			_, _ = analyzer.Run(pass)

			for _, v := range diag {
				pos := getPosition(pkg.Fset, v.Pos)
				msg := fmt.Sprintf("%v: %s", pos, v.Message)
				if slices.Index(messages, msg) == -1 {
					mu.Lock()
					messages = append(messages, msg)
					mu.Unlock()
				}
			}
		}()
	}
	wg.Wait()
	return messages
}
