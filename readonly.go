package readonly

import (
	"go/ast"
	"go/token"
	"go/types"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/packages"
)

//func New(ro *config.ReadonlySettings) *goanalysis.Linter {
//	a := NewAnalyzer()
//	return goanalysis.NewLinter(a.Name, a.Doc, []*analysis.Analyzer{a}, nil).WithContextSetter(func(context *linter.Context) {
//		initTypes(context.Packages)
//	}).WithLoadMode(goanalysis.LoadModeTypesInfo)
//}

func NewAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: linterName,
		Doc:  docEN,
		Run:  runAnalyzer,
	}
}

func Setup(pkgs []*packages.Package) {
	initTypes(pkgs)
}

func CheckDir(dir string) {
	checkDir(dir)
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

func initTypes(pkgs []*packages.Package) {
	allPackages = pkgs
	funcTypes = make(map[token.Pos]*funcInfo)
	moduleDir = pkgs[0].Module.Dir

	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			for _, decl := range file.Decls {
				if fd, ok := decl.(*ast.FuncDecl); ok {
					object := pkg.TypesInfo.Defs[fd.Name].(*types.Func)
					info := &funcInfo{decl: fd, fullName: object.FullName()}
					info.calcDecl()
					funcTypes[fd.Name.Pos()] = info
				}
			}
		}
	}

	for i := 0; i < 100; i++ {
		changed := false
		for _, info := range funcTypes {
			old := info.roMask
			info.calcBody()
			if old != info.roMask {
				changed = true
			}
		}
		if !changed {
			return
		}
	}
}

func runAnalyzer(pass *analysis.Pass) (interface{}, error) {
	for _, f := range pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			if decl, ok := n.(*ast.FuncDecl); ok {
				for _, stmt := range decl.Body.List {
					checkStmt(pass, stmt)
				}
			}
			return true
		})
	}
	return nil, nil
}
