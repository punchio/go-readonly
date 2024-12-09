package readonly

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"go/types"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/packages"
	"log"
	"path/filepath"
	"slices"
)

func checkDir(dir string) {
	// 配置解析选项
	cfg := &packages.Config{
		Mode:  packages.NeedName | packages.NeedSyntax | packages.NeedTypesInfo | packages.NeedModule,
		Dir:   dir,   // 指定项目目录
		Tests: false, // 不包含测试文件
	}

	// 加载所有 package
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		panic(fmt.Sprintf("loading packages fail, err: %v", err))
		return
	}
	initTypes(pkgs)

	analyzer := &analysis.Analyzer{
		Name: linterName,
		Doc:  "check variable with 'ro' start", // 文档说明
		Run:  runAnalyzer,                      // 执行分析的函数
	}

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

		var messages []string
		for _, v := range diag {
			pos := getPosition(pkg.Fset, v.Pos)
			msg := fmt.Sprintf("%v: %s", pos, v.Message)
			if slices.Index(messages, msg) == -1 {
				messages = append(messages, msg)
			}
		}
		for _, v := range messages {
			log.Println(v)
		}
	}
}

func checkStmt(pass *analysis.Pass, stmt ast.Stmt) {
	fset := pass.Fset
	ast.Inspect(stmt, func(node ast.Node) bool {
		if expr, ok := node.(ast.Expr); ok {
			checkExpr(pass, expr)
		}
		return true
	})
	switch e := stmt.(type) {
	case *ast.IncDecStmt:
		if getExprReadonlyFlag(e.X) > 0 {
			var buf bytes.Buffer
			_ = printer.Fprint(&buf, fset, e.X)
			pass.Reportf(e.X.Pos(), `variable "%s" cannot be modify`, buf.String())
		}
	case *ast.AssignStmt:
		assign, lhs, rhs := collectAssignStmt(e)
		lhsFlag, skipFlag := collectLhsFlag(lhs)
		rhsFlag := collectRhsFlag(rhs, skipFlag)
		checkAssign(assign, lhs, rhs, lhsFlag, rhsFlag, skipFlag, pass)
	case *ast.DeclStmt:
		decl := e.Decl.(*ast.GenDecl)
		for _, spec := range decl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			assign, lhs, rhs := collectValueSpec(valueSpec)
			lhsFlag, skipFlag := collectLhsFlag(lhs)
			rhsFlag := collectRhsFlag(rhs, skipFlag)
			checkAssign(assign, lhs, rhs, lhsFlag, rhsFlag, skipFlag, pass)
		}
	case *ast.RangeStmt:
		var lhs, rhs []ast.Expr
		lhs = append(lhs, e.Key, e.Value)
		rhs = append(rhs, e.X)
		lhsFlag, skipFlag := collectLhsFlag(lhs)
		rhsFlag := collectRhsFlag(rhs, skipFlag)
		rhsFlag |= rhsFlag << 1
		checkAssign(false, lhs, rhs, lhsFlag, rhsFlag, skipFlag, pass)
	}
}

func checkExpr(pass *analysis.Pass, expr ast.Expr) {
	fset := pass.Fset
	roFlag := false
	unwrapCheckExpr(expr, func(expr ast.Expr) {
		if getExprReadonlyFlag(expr) > 0 {
			roFlag = true
		} else if roFlag {
			ident, ok := expr.(*ast.Ident)
			if !ok {
				return
			}
			_, ok = funcTypes[getNamePos(ident)]
			if !ok {
				return
			}
			var buf bytes.Buffer
			_ = printer.Fprint(&buf, fset, expr)
			pass.Reportf(expr.Pos(), `variable "%s" cannot call non readonly method`, buf.String())
		}
	})
}

func checkAssign(isAssign bool, lhs, rhs []ast.Expr, lhsFlag, rhsFlag uint64, skipFlag uint64, pass *analysis.Pass) {
	fset := pass.Fset
	skipIndexes := make([]bool, len(lhs))
	for i := range lhs {
		if skipFlag&(1<<i) != 0 {
			skipIndexes[i] = true
		}
	}

	if isAssign {
		for i := 0; i < len(lhs); i++ {
			if skipIndexes[i] {
				continue
			}

			if lhsFlag&(1<<i) != 0 {
				skipIndexes[i] = true
				var buf bytes.Buffer
				_ = printer.Fprint(&buf, fset, lhs[i])
				pass.Reportf(lhs[i].Pos(), `variable "%s" cannot be assigned`, buf.String())
			}
		}
	}

	for i := 0; i < len(lhs); i++ {
		if skipIndexes[i] {
			continue
		}

		// 如果右值为非只读，或者左值是只读，则跳过
		if rhsFlag&(1<<i) == 0 || lhsFlag&(1<<i) != 0 {
			continue
		}

		var rhsBuf, lhsBuf bytes.Buffer
		_ = printer.Fprint(&lhsBuf, fset, lhs[i])
		var rhsPos token.Pos
		if len(rhs) == len(lhs) {
			rhsPos = rhs[i].Pos()
			_ = printer.Fprint(&rhsBuf, fset, rhs[i])
		} else {
			rhsPos = rhs[0].Pos()
			_ = printer.Fprint(&rhsBuf, fset, rhs[0])
		}

		pass.Reportf(lhs[i].Pos(), `variable "%s" cannot assigned with 
	readonly variable "%s" at %v `, lhsBuf.String(), rhsBuf.String(), getPosition(fset, rhsPos))
	}
}

// collectLhsFlag 获取左值的只读标记和跳过标记
// 左值和右值不一样的地方在于，左值类型限制更严格，右值更随意一些
func collectLhsFlag(lhs []ast.Expr) (roFlag uint64, skipFlag uint64) {
	/*
		左值可能为这些情况
			变量	*ast.Ident	a = 10
			指针解引用	*ast.StarExpr	*p = 20
			数组或切片的元素	*ast.IndexExpr	arr[0] = 5
			结构体字段	*ast.SelectorExpr	person.name = "John"
			字典（映射）元素	*ast.IndexExpr	m["key"] = "value"
			接口变量的底层实现字段	*ast.SelectorExpr	w.Write([]byte("Hello"))
	*/
	for i, expr := range lhs {
		if ident, ok := expr.(*ast.Ident); ok && ident.Name == "_" {
			skipFlag |= 1 << i
			continue
		}

		t := getExprType(expr)
		if _, ok := t.(*types.Basic); ok {
			skipFlag |= 1 << i
			continue
		}

		tmp := getExprReadonlyFlag(expr)
		if tmp > 0 {
			roFlag |= 1 << i
		}
	}
	return
}

func collectRhsFlag(rhs []ast.Expr, skipFlag uint64) uint64 {
	flag := uint64(0)
	for i, expr := range rhs {
		if skipFlag&(1<<i) != 0 {
			continue
		}
		tmp := getExprReadonlyFlag(expr)
		if tmp > 0 {
			flag |= 1 << i
		}
	}
	return flag
}

func collectAssignStmt(stmt *ast.AssignStmt) (isAssign bool, lhs []ast.Expr, rhs []ast.Expr) {
	return stmt.Tok == token.ASSIGN, stmt.Lhs, stmt.Rhs
}

func collectValueSpec(spec *ast.ValueSpec) (isAssign bool, lhs []ast.Expr, rhs []ast.Expr) {
	isAssign = false
	for _, v := range spec.Names {
		lhs = append(lhs, v)
	}
	rhs = spec.Values
	return
}

func unwrapCheckExpr(expr ast.Expr, f func(ast.Expr)) {
	cur := expr
	for cur != nil {
		switch e := cur.(type) {
		case *ast.Ident:
			f(e)
			cur = nil
		case *ast.StarExpr:
			cur = e.X
		case *ast.ParenExpr:
			cur = e.X
		case *ast.UnaryExpr:
			cur = e.X
		case *ast.IndexExpr:
			unwrapCheckExpr(e.Index, f)
			cur = e.X
		case *ast.IndexListExpr:
			for _, v := range e.Indices {
				unwrapCheckExpr(v, f)
			}
			cur = e.X
		case *ast.CallExpr:
			for _, v := range e.Args {
				unwrapCheckExpr(v, f)
			}
			unwrapCheckExpr(e.Fun, f)
			cur = nil
		case *ast.SelectorExpr:
			unwrapCheckExpr(e.X, f)
			f(e.Sel)
			cur = nil
		default:
			cur = nil
		}
		if cur != nil {
			f(cur)
		}
	}
}

func getExprReadonlyFlag(expr ast.Expr) uint64 {
	switch t := expr.(type) {
	case *ast.Ident:
		if checkName(t.Name) {
			return 1
		}
		pos := getNamePos(t)
		if info, ok := funcTypes[pos]; ok {
			return info.getResultFlag()
		}
		return 0
	case *ast.SelectorExpr:
		return getSelectorRoFlag(t)
	case *ast.CallExpr:
		return getRoFuncResultFlag(t)
	case *ast.StarExpr:
		return getExprReadonlyFlag(t.X)
	case *ast.IndexExpr:
		return getExprReadonlyFlag(t.X)
	case *ast.IndexListExpr:
		return getExprReadonlyFlag(t.X)
	case *ast.UnaryExpr:
		return getExprReadonlyFlag(t.X)
	default:
		return 0
	}
}

func getSelectorRoFlag(sel *ast.SelectorExpr) uint64 {
	flag := getExprReadonlyFlag(sel.Sel)
	if flag > 0 {
		return flag
	}
	flag = getExprReadonlyFlag(sel.X)
	if flag > 0 {
		return 1
	}
	return 0
}

func getRoFuncResultFlag(call *ast.CallExpr) uint64 {
	ident, ok := call.Fun.(*ast.Ident)
	if ok {
		pos := getNamePos(ident)
		info, ok := funcTypes[pos]
		// 可能是类型强转
		if !ok {
			return 0
		}
		return info.getResultFlag()
	}
	return getExprReadonlyFlag(call.Fun)
}

func getNamePos(ident *ast.Ident) token.Pos {
	for _, p := range allPackages {
		if obj, ok := p.TypesInfo.Uses[ident]; ok {
			return obj.Pos()
		}
		if obj, ok := p.TypesInfo.Defs[ident]; ok && obj != nil {
			return obj.Pos()
		}
	}

	return token.NoPos
}

func getExprType(expr ast.Expr) types.Type {
	for _, p := range allPackages {
		if obj, ok := p.TypesInfo.Types[expr]; ok {
			return obj.Type
		}
	}
	return nil
}
func checkName(name string) bool {
	l := len(roPrefix)
	return name == roPrefix ||
		(len(name) > l && name[:l] == roPrefix && name[l] >= 'A' && name[l] <= 'Z')
}

func getPosition(fset *token.FileSet, pos token.Pos) token.Position {
	p := fset.Position(pos)
	shortPath, _ := filepath.Rel(moduleDir, p.Filename)
	p.Filename = shortPath
	return p
}
