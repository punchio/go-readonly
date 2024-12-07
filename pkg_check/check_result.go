package pkg_check

import (
	"go/ast"
	"go/token"
	"go/types"
	"golang.org/x/tools/go/packages"
)

const roPrefix = "ro"
const roLen = len(roPrefix)

func loadPackages(pkg *packages.Package) {
	for _, syntax := range pkg.Syntax {
		for _, decl := range syntax.Decls {
			if fd, ok := decl.(*ast.FuncDecl); ok {
				object := pkg.TypesInfo.Defs[fd.Name].(*types.Func)
				info := &funcInfo{decl: fd, fullName: object.FullName(), pkg: pkg}
				info.calcMask(true)
				if funcTypes[fd.Name.Pos()] != nil {
					panic("duplicate func name")
				}
				funcTypes[fd.Name.Pos()] = info
			}
		}
	}
	allPackage = append(allPackage, pkg)
}
func fixFuncRoMask() {
	changed := true
	maxCount := 1000
	for changed {
		maxCount--
		if maxCount <= 0 {
			panic("func ro mask not stable")
		}
		changed = false
		for _, info := range funcTypes {
			old := info.roMask
			info.calcMask(false)
			if old != info.roMask {
				changed = true
			}
		}
	}
}
func calcFuncMask(decl *ast.FuncDecl, body *ast.BlockStmt) uint64 {
	roMask := uint64(0)

	funcType := decl.Type
	if decl.Recv != nil && len(decl.Recv.List) > 0 &&
		len(decl.Recv.List[0].Names) > 0 &&
		checkName(decl.Recv.List[0].Names[0].Name) {
		roMask = 1 << 63
	}
	if funcType.Params != nil {
		index := 0
		for _, field := range funcType.Params.List {
			for _, name := range field.Names {
				if checkName(name.Name) {
					roMask |= 1 << index
				}
				index++
			}
		}
	}

	// 检测命名返回值中的只读
	if funcType.Results != nil {
		index := 32
		for _, field := range funcType.Results.List {
			for _, name := range field.Names {
				if checkName(name.Name) {
					roMask |= 1 << index
				}
				index++
			}
		}
	}

	if body != nil && funcType.Results != nil {
		// 检测返回语句中的只读变量
		index := 32
		for _, stmt := range body.List {
			returnStmt, ok := stmt.(*ast.ReturnStmt)
			if !ok {
				continue
			}

			// 函数有多个返回值，但是返回语句表达式只有一个，则肯定是函数调用返回
			if len(returnStmt.Results) == 1 && len(funcType.Results.List) > 1 {
				call := returnStmt.Results[0].(*ast.CallExpr)
				roMask |= getRoFuncResultFlag(call) << index
			} else {
				// 对每个返回值单独判断是否只读
				for _, expr := range returnStmt.Results {
					if ident, ok := expr.(*ast.Ident); ok {
						if checkName(ident.Name) {
							roMask |= 1 << index
						}
					} else if call, ok := expr.(*ast.CallExpr); ok {
						if getRoFuncResultFlag(call) > 0 {
							roMask |= 1 << index
						}
					} else if sel, ok := expr.(*ast.SelectorExpr); ok {
						if getSelectorRoFlag(sel) > 0 {
							roMask |= 1 << index
						}
					}
					index++
				}
			}
		}
	}
	return roMask
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
	for _, p := range allPackage {
		if obj, ok := p.TypesInfo.Uses[ident]; ok {
			return obj.Pos()
		}
		if obj, ok := p.TypesInfo.Defs[ident]; ok {
			return obj.Pos()
		}
	}
	return token.NoPos
}

func getExprType(expr ast.Expr) types.Type {
	for _, p := range allPackage {
		if obj, ok := p.TypesInfo.Types[expr]; ok {
			return obj.Type
		}
	}
	return nil
}
func checkName(name string) bool {
	return name == roPrefix ||
		(len(name) > roLen &&
			name[:roLen] == roPrefix &&
			name[roLen] >= 'A' && name[roLen] <= 'Z')
}
