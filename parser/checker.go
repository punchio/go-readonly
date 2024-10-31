/*
变量初始化、赋值两种情况的检查
基础类型：
	1. 如果变量的类型为基础类型，不用检查其对应的只读类型
	2. _ 变量不用检查，可以用在多值返回时，不需要只读变量的情况
数量：
	1. 多左值，多右值
		必然左值和右值数量相等，且右值不能有多返回值函数
	2. 多左值，单右值
		必然是右值为多返回值函数或者方法
左值变量类型区别：
	1. 赋值可以为结构体变量的字段赋值，初始化声明只能是新增变量
只读限制区别：
	1. 只读字段不能再被赋值，所以赋值语句中左值中有只读变量都是错误；所以右值也不能有只读变量
	2. 初始化时，右值为只读时，左值也需要为只读变量；右值不为只读，左值没限制；所以，只用检查右值只读变量对应的左值即可
需要检查的语句：
	1. ast.AssignStmt 赋值语句，要区分对待 := 初始化语句和 = 赋值语句
	2. ast.DeclStmt 声明语句
		ast.GenDecl的Specs可以为多个，每一个ast.ValueSpec的判断跟一次 := 流程一样

检查策略
	1. 收集左值的只读属性
	2. 收集右值的只读属性，如果右值为基础类型，即使是只读，也标记为非只读
	3. 根据语句类型，初始化or赋值，比较左右值集合是否满足
*/

package parser

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"go/types"
	"log"
	"math"
	"readonly/tools"
	"strings"
)

const roPrefix = "ro"

func check(isAssign bool, lhs, rhs []ast.Expr, lhsFlag, rhsFlag uint64, skipFlag uint64, fset *token.FileSet) {
	skipIndexes := make([]bool, len(lhs))
	for i := 0; i < len(lhs); i++ {
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
				log.Printf("[lhs:%s,at %v ] cannot be assigned\n", buf.String(), fset.Position(lhs[i].Pos()))
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
		var pos token.Pos
		if len(rhs) == len(lhs) {
			pos = rhs[i].Pos()
			_ = printer.Fprint(&rhsBuf, fset, rhs[i])
		} else {
			pos = rhs[0].Pos()
			_ = printer.Fprint(&rhsBuf, fset, rhs[0])
		}

		log.Printf("variable [lhs:%s at %v ] cannot assigned with \n\t\t\t\tvariable readonly [rhs:%s at %v ] ",
			lhsBuf.String(), fset.Position(lhs[i].Pos()),
			rhsBuf.String(), fset.Position(pos))
	}
}

// collectLhsFlag 获取左值的只读标记和跳过标记
// 左值和右值不一样的地方在于，左值类型限制更严格，右值更随意一些
func collectLhsFlag(lhs []ast.Expr, fset *token.FileSet) (roFlag uint64, skipFlag uint64) {
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
		switch t := expr.(type) {
		case *ast.Ident:
			// 需要对 _ 特殊处理
			if t.Name == "_" {
				skipFlag |= 1 << i
				continue
			}

			varType := getVarType(t)
			if _, ok := varType.(*ast.BasicLit); ok {
				skipFlag |= 1 << i
				continue
			}

			tmp := getExprReadonlyFlag(expr)
			if tmp > 0 {
				roFlag |= 1 << i
			}
		case *ast.StarExpr,
			*ast.IndexExpr,
			*ast.SelectorExpr:
			tmp := getExprReadonlyFlag(expr)
			if tmp > 0 {
				roFlag |= 1 << i
			}
		default:
			text := tools.GetExprText(fset, expr)
			panic(fmt.Sprintf("lhs expr illegal, expr:%s", text))
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

func checkReadonly(fset *token.FileSet) error {
	for _, v := range funcInfos {
		for i, stmt := range v.decl.Body.List {
			_ = i
			ast.Inspect(stmt, func(node ast.Node) bool {
				switch e := node.(type) {
				case *ast.CallExpr:
					// 检查函数调用参数传入，普通变量可以传入只读变量参数，只读变量不能传入普通参数，变量可看作右值，参数可看作左值，与声明一样
					if e.Args == nil {
						break
					}
					rhs := e.Args
					rhsFlag := collectRhsFlag(rhs, 0)

					lhs := make([]ast.Expr, 0, len(rhs))
					for i := 0; i < len(rhs); i++ {
						lhs = append(lhs, e)
					}
					lhsFlag := getRoFuncParamFlag(e)
					check(false, lhs, rhs, lhsFlag, rhsFlag, 0, fset)
				case *ast.AssignStmt:
					assign, lhs, rhs := collectAssignStmt(e)
					lhsFlag, skipFlag := collectLhsFlag(lhs, fset)
					rhsFlag := collectRhsFlag(rhs, skipFlag)
					check(assign, lhs, rhs, lhsFlag, rhsFlag, skipFlag, fset)
				case *ast.DeclStmt:
					decl := e.Decl.(*ast.GenDecl)
					for _, spec := range decl.Specs {
						valueSpec := spec.(*ast.ValueSpec)
						assign, lhs, rhs := collectValueSpec(valueSpec)
						lhsFlag, skipFlag := collectLhsFlag(lhs, fset)
						rhsFlag := collectRhsFlag(rhs, skipFlag)
						check(assign, lhs, rhs, lhsFlag, rhsFlag, skipFlag, fset)
					}
				case *ast.RangeStmt:
					typ := getVarType(e.X)

					var lhs, rhs []ast.Expr
					if _, ok := typ.(*ast.MapType); ok {
						lhs = append(lhs, e.Key, e.Value)
					} else if _, ok := typ.(*ast.ArrayType); ok {
						lhs = append(lhs, e.Value)
					} else {
						panic(fmt.Sprintf("unsupported range type,text:%s", tools.GetExprText(fset, e.X)))
					}
					rhs = append(rhs, e.X)
					lhsFlag, skipFlag := collectLhsFlag(lhs, fset)
					rhsFlag := collectRhsFlag(rhs, skipFlag)
					rhsFlag |= rhsFlag << 1
					check(false, lhs, rhs, lhsFlag, rhsFlag, skipFlag, fset)
				}
				return true
			})
		}
	}
	return nil
}

//
//func getExprDeclType(expr ast.Expr) []*ast.TypeSpec {
//	switch t := expr.(type) {
//	case *ast.Ident:
//		if t.Obj.Kind == ast.Typ {
//			ts := t.Obj.Decl.(*ast.TypeSpec)
//			return []*ast.TypeSpec{ts}
//		} else if t.Obj.Kind == ast.Fun {
//			fd := t.Obj.Decl.(*ast.FuncDecl)
//			var funcResult []*ast.TypeSpec
//			for _, v := range fd.Type.Results.List {
//				ident := getStarIdent(v.Type)
//				ts := ident.Obj.Decl.(*ast.TypeSpec)
//				nameCount := len(v.Names)
//				if nameCount == 0 {
//					nameCount = 1
//				}
//				for i := 0; i < nameCount; i++ {
//					funcResult = append(funcResult, ts)
//				}
//			}
//			return funcResult
//		} else if t.Obj.Kind == ast.Var {
//			f, ok := t.Obj.Decl.(*ast.Field)
//			if ok {
//				return getExprDeclType(f.Type)
//			}
//
//			assignStmt := t.Obj.Decl.(*ast.AssignStmt)
//			for i, lhs := range assignStmt.Lhs {
//				if lhs == expr {
//					switch t := assignStmt.Rhs[i].(type) {
//					case *ast.BasicLit:
//						return []*ast.TypeSpec{nil}
//					case *ast.CompositeLit:
//						return getExprDeclType(t.Type)
//					case *ast.CallExpr:
//						return getExprDeclType(t.Fun)
//					case *ast.SelectorExpr:
//						return getExprDeclType(t.X)
//					case *ast.IndexExpr:
//						return getExprDeclType(t.X)
//					case *ast.IndexListExpr:
//						return getExprDeclType(t.X)
//					case *ast.UnaryExpr:
//						if t.Op == token.RANGE {
//							var ts []*ast.TypeSpec
//							ast.Inspect(t.X, func(node ast.Node) bool {
//								switch t := node.(type) {
//								case *ast.MapType:
//									if i == 0 {
//										ts = append(ts, getExprDeclType(t.Key)...)
//									} else {
//										ts = append(ts, getExprDeclType(t.Value)...)
//									}
//									return false
//								case *ast.ArrayType:
//									ts = append(ts, getExprDeclType(t.Elt)...)
//									return false
//								}
//								return true
//							})
//						}
//						return getExprDeclType(t.X)
//					case *ast.BinaryExpr:
//						return getExprDeclType(t.X)
//					}
//					break
//				}
//			}
//		}
//	case *ast.CallExpr:
//		return getExprDeclType(t.Fun)
//	case *ast.SelectorExpr:
//		return getExprDeclType(t.X)
//	case *ast.StarExpr:
//		return getExprDeclType(t.X)
//	case *ast.CompositeLit:
//		return getExprDeclType(t.Type)
//	case *ast.UnaryExpr:
//		return getExprDeclType(t.X)
//	case *ast.BinaryExpr:
//		return getExprDeclType(t.X)
//	case *ast.BasicLit:
//		return []*ast.TypeSpec{nil}
//	case *ast.IndexExpr:
//		return getExprDeclType(t.X)
//	case *ast.IndexListExpr:
//		return getExprDeclType(t.X)
//	}
//	return nil
//}

func getExprReadonlyFlag(expr ast.Expr) uint64 {
	switch t := expr.(type) {
	case *ast.Ident:
		if checkName(t.Name) {
			return 1
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

func checkName(name string) bool {
	return strings.HasPrefix(name, roPrefix)
}

func getRoFuncResultFlag(call *ast.CallExpr) uint64 {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if ok {
		return getSelectorRoFlag(sel)
	}
	ident := call.Fun.(*ast.Ident)
	if ident.Obj == nil {
		return 0
	}

	decl, ok := ident.Obj.Decl.(*ast.FuncDecl)
	if !ok { // 可能为强制转换
		return 0
	}
	info := funcInfos[decl]
	return info.getRoResultFlag()
}

func getRoFuncParamFlag(call *ast.CallExpr) uint64 {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if ok {
		return getSelectorRoFlag(sel)
	}
	ident := call.Fun.(*ast.Ident)
	if ident.Obj == nil {
		return 0
	}
	decl, ok := ident.Obj.Decl.(*ast.FuncDecl)
	if !ok { // 可能为强制转换
		return 0
	}
	info := funcInfos[decl]
	return info.getRoParamFlag()
}

func getSelectorRoFlag(sel *ast.SelectorExpr) uint64 {
	// 获取选择器的第一个标识符
	ident, names := getSelectorSequence(sel)
	if ident.Obj.Kind == ast.Var && checkName(ident.Name) {
		return 1
	}

	expr := getVarType(ident)
	for i, name := range names {
		if tmp, ok := expr.(*ast.Ident); ok {
			expr = getVarType(tmp.Obj.Decl.(*ast.TypeSpec).Type)
		}

		switch t := expr.(type) {
		case *ast.ArrayType:
			expr = getVarType(t.Elt)
		case *ast.MapType:
			expr = getVarType(t.Value)
		case *ast.StructType:
			info, ok := structInfos[t]
			if !ok {
				panic(fmt.Sprintf("getSelectorTypes fail, expr:%s", types.ExprString(expr)))
			}
			member, method := info.getMember(name)
			if member != nil {
				expr = getVarType(member)
			} else {
				expr = getVarType(method.getDecl())
				roFlag := method.getRoResultFlag()
				if roFlag > 0 {
					if i < len(names)-1 {
						return math.MaxUint64
					} else {
						return roFlag
					}
				}
			}
		case *ast.InterfaceType:
			for _, field := range t.Methods.List {
				if field.Names[0].Name == name {
					expr = getVarType(field.Type)
					break
				}
			}
		case *ast.FuncType:
			info, ok := funcTypeInfos[t]
			if !ok {
				panic(fmt.Sprintf("getSelectorTypes fail, expr:%s", types.ExprString(expr)))
			}
			roFlag := info.getRoResultFlag()
			if roFlag > 0 {
				if i < len(names)-1 {
					return math.MaxUint64
				} else {
					return roFlag
				}
			}
			if t.Results != nil {
				expr = getVarType(t.Results.List[0].Type)
			} else {
				expr = nil
			}
		case *ast.FuncLit:
			roFlag := calcFuncResultMask(t.Type, t.Body)
			if roFlag > 0 {
				if i < len(names)-1 {
					return math.MaxUint64
				} else {
					return roFlag
				}
			}
			return 0
		default:
			panic(fmt.Sprintf("getSelectorTypes fail, expr:%s", types.ExprString(expr)))
		}
	}
	return 0
}
