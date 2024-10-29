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
	"slices"
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
func collectLhsFlag(lhs []ast.Expr, fset *token.FileSet, info *types.Info) (roFlag uint64, skipFlag uint64) {
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

			if obj := info.Defs[t]; obj != nil {
				if _, ok := obj.Type().(*types.Basic); ok {
					skipFlag |= 1 << i
					continue
				}
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

func checkReadonly(file *ast.File, fset *token.FileSet, info *types.Info) error {
	for _, v := range funcInfos {
		for _, stmt := range v.decl.Body.List {
			_ = stmt
			switch e := stmt.(type) {
			case *ast.AssignStmt:
				assign, lhs, rhs := collectAssignStmt(e)
				lhsFlag, skipFlag := collectLhsFlag(lhs, fset, info)
				rhsFlag := collectRhsFlag(rhs, skipFlag)
				check(assign, lhs, rhs, lhsFlag, rhsFlag, skipFlag, fset)
			case *ast.DeclStmt:
				decl := e.Decl.(*ast.GenDecl)
				for _, spec := range decl.Specs {
					valueSpec := spec.(*ast.ValueSpec)
					assign, lhs, rhs := collectValueSpec(valueSpec)
					lhsFlag, skipFlag := collectLhsFlag(lhs, fset, info)
					rhsFlag := collectRhsFlag(rhs, skipFlag)
					check(assign, lhs, rhs, lhsFlag, rhsFlag, skipFlag, fset)
				}
			case *ast.RangeStmt:
				typ, ok := info.Types[e.X]
				if !ok {
					panic(fmt.Sprintf("not found type,text:%s", tools.GetExprText(fset, e.X)))
				}

				var lhs, rhs []ast.Expr
				if _, ok = typ.Type.(*types.Map); ok {
					lhs = append(lhs, e.Key, e.Value)
				} else if _, ok = typ.Type.(*types.Slice); ok {
					lhs = append(lhs, e.Value)
				} else {
					panic(fmt.Sprintf("unsupported range type,text:%s", tools.GetExprText(fset, e.X)))
				}
				rhs = append(rhs, e.X)
				lhsFlag, skipFlag := collectLhsFlag(lhs, fset, info)
				rhsFlag := collectRhsFlag(rhs, skipFlag)
				rhsFlag |= rhsFlag << 1
				check(false, lhs, rhs, lhsFlag, rhsFlag, skipFlag, fset)
			}
		}
	}
	return nil
}

func getSelectorTree(sel *ast.SelectorExpr) (*ast.Ident, []string) {
	var sels = []string{sel.Sel.Name}
	cur := sel.X
	var root *ast.Ident
LOOP:
	for {
		switch e := cur.(type) {
		case *ast.CallExpr:
			cur = e.Fun
		case *ast.SelectorExpr:
			cur = e.X
			sels = append(sels, e.Sel.Name)
		case *ast.Ident:
			root = e
			break LOOP
		default:
			panic("unsupported call expr")
		}
	}
	slices.Reverse(sels)
	return root, sels
}

func getExprDeclType(expr ast.Expr) []*ast.TypeSpec {
	switch t := expr.(type) {
	case *ast.Ident:
		if t.Obj.Kind == ast.Typ {
			ts := t.Obj.Decl.(*ast.TypeSpec)
			return []*ast.TypeSpec{ts}
		} else if t.Obj.Kind == ast.Fun {
			fd := t.Obj.Decl.(*ast.FuncDecl)
			var funcResult []*ast.TypeSpec
			for _, v := range fd.Type.Results.List {
				ident := getFinalIdent(v.Type)
				ts := ident.Obj.Decl.(*ast.TypeSpec)
				nameCount := len(v.Names)
				if nameCount == 0 {
					nameCount = 1
				}
				for i := 0; i < nameCount; i++ {
					funcResult = append(funcResult, ts)
				}
			}
			return funcResult
		} else if t.Obj.Kind == ast.Var {
			f, ok := t.Obj.Decl.(*ast.Field)
			if ok {
				return getExprDeclType(f.Type)
			}

			assignStmt := t.Obj.Decl.(*ast.AssignStmt)
			for i, lhs := range assignStmt.Lhs {
				if lhs == expr {
					switch t := assignStmt.Rhs[i].(type) {
					case *ast.BasicLit:
						return []*ast.TypeSpec{nil}
					case *ast.CompositeLit:
						return getExprDeclType(t.Type)
					case *ast.CallExpr:
						return getExprDeclType(t.Fun)
					case *ast.SelectorExpr:
						return getExprDeclType(t.X)
					case *ast.IndexExpr:
						return getExprDeclType(t.X)
					case *ast.IndexListExpr:
						return getExprDeclType(t.X)
					case *ast.UnaryExpr:
						if t.Op == token.RANGE {
							var ts []*ast.TypeSpec
							ast.Inspect(t.X, func(node ast.Node) bool {
								switch t := node.(type) {
								case *ast.MapType:
									if i == 0 {
										ts = append(ts, getExprDeclType(t.Key)...)
									} else {
										ts = append(ts, getExprDeclType(t.Value)...)
									}
									return false
								case *ast.ArrayType:
									ts = append(ts, getExprDeclType(t.Elt)...)
									return false
								}
								return true
							})
						}
						return getExprDeclType(t.X)
					case *ast.BinaryExpr:
						return getExprDeclType(t.X)
					}
					break
				}
			}
		}
	case *ast.CallExpr:
		return getExprDeclType(t.Fun)
	case *ast.SelectorExpr:
		return getExprDeclType(t.X)
	case *ast.StarExpr:
		return getExprDeclType(t.X)
	case *ast.CompositeLit:
		return getExprDeclType(t.Type)
	case *ast.UnaryExpr:
		return getExprDeclType(t.X)
	case *ast.BinaryExpr:
		return getExprDeclType(t.X)
	case *ast.BasicLit:
		return []*ast.TypeSpec{nil}
	case *ast.IndexExpr:
		return getExprDeclType(t.X)
	case *ast.IndexListExpr:
		return getExprDeclType(t.X)
	}
	return nil
}

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
		return getRoFuncFlag(t)
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

func getRoFuncFlag(call *ast.CallExpr) uint64 {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if ok {
		return getSelectorRoFlag(sel)
	}
	ident := call.Fun.(*ast.Ident)
	decl := ident.Obj.Decl.(*ast.FuncDecl)
	info := funcInfos[decl]
	return info.getRoResultFlag()
}

func getIdentTypeIdent(ident *ast.Ident) *ast.Ident {
	if ident.Obj.Kind == ast.Typ {
		switch t := ident.Obj.Decl.(type) {
		case *ast.TypeSpec:
			return t.Name
		case *ast.Field:
			return getFinalIdent(t.Type)
		case *ast.ImportSpec:
			return t.Name
		default:
			panic("type ident not found")
		}
	} else if ident.Obj.Kind == ast.Fun {
		d := ident.Obj.Decl.(*ast.FuncDecl)
		return d.Name
	} else if ident.Obj.Kind == ast.Con {
		vs := ident.Obj.Decl.(*ast.ValueSpec)
		for i, name := range vs.Names {
			if ident.Name == name.Name {
				if len(vs.Names) != len(vs.Values) {
					result := vs.Values[i].(*ast.Ident)
					return result
				} else {
					result := vs.Values[i].(*ast.Ident)
					return result
				}
			}
		}
	} else if ident.Obj.Kind == ast.Var {
		switch t := ident.Obj.Decl.(type) {
		case *ast.AssignStmt:
			if len(t.Lhs) == len(t.Rhs) {
				for i, lhs := range t.Lhs {
					tmp := lhs.(*ast.Ident)
					if tmp.Name == ident.Name {
						ident = getIdentType(t.Rhs[i])
						return getIdentTypeIdent(ident)
					}
				}
			} else {

			}
		case *ast.Field: // 方法结构体参数
			ident = getIdentType(t.Type)
			return getIdentTypeIdent(ident)
		case *ast.CompositeLit:
			ident = getIdentType(t.Type)
			return getIdentTypeIdent(ident)
		}
	}
	panic("ident not found")
}

func getIdentType(expr ast.Expr) *ast.Ident {
	switch t := expr.(type) {
	case *ast.CompositeLit:
		return t.Type.(*ast.Ident)
	case *ast.SelectorExpr:
		ident, sels := getSelectorTree(t)

	}
	panic("expr not found")
}

func getSelectorRoFlag(sel *ast.SelectorExpr) uint64 {
	// 获取选择器的第一个标识符
	ident, sels := getSelectorTree(sel)
	ident = getIdentTypeIdent(ident)
	switch t := ident.Obj.Decl.(type) {
	case *ast.AssignStmt: // 临时变量
		for i, lhs := range t.Lhs {
			tmp := lhs.(*ast.Ident)
			if tmp.Name == ident.Name {
				ts := getExprDeclType(t.Rhs[i])
				ident = ts[0].Name
				break
			}
		}
	case *ast.Field: // 方法结构体参数
		ident = getFinalIdent(t.Type)
	case *ast.ValueSpec:
		for i, name := range t.Names {
			if name.Name == ident.Name {
				//t.Values[i]
				_ = i
			}
		}
	default:
		panic("selector type not found")
	}
	// 最后一次选择函数可能返回多个值
	cur := ident
	for i := 0; i < len(sels); i++ {
		var curTypeSpec *ast.TypeSpec
		switch cur.Obj.Kind {
		case ast.Typ: // 对应值类型
			curTypeSpec = cur.Obj.Decl.(*ast.TypeSpec)
		case ast.Fun: // 第一次出现是函数，之后都是方法
			decl := cur.Obj.Decl.(*ast.FuncDecl)
			tmp := getFinalIdent(decl.Type.Results.List[0].Type)
			curTypeSpec = tmp.Obj.Decl.(*ast.TypeSpec)
		case ast.Var: // 对应结构体字段
			field := cur.Obj.Decl.(*ast.Field)
			tmp := getFinalIdent(field.Type)
			curTypeSpec = tmp.Obj.Decl.(*ast.TypeSpec)
		default:
			panic("unsupported selector")
		}

		info := typeInfos[curTypeSpec]
		member, method := info.getMember(sels[i])
		if member == nil && method == nil {
			panic(fmt.Sprintf("struct member not found, member:%s,struct:%v", sels[i], cur.Obj))
		}

		// 如果是调用过程中返回只读数据，则整个表达式的结构都是只读
		// 有可能最后一个调用返回多个结果
		if i < len(sels)-1 {
			if member != nil {
				cur = member
				if strings.HasPrefix(cur.Name, roPrefix) {
					return math.MaxUint64
				}
			} else if method != nil {
				cur = method.getIdent()
				if method.isRoResult(0) {
					return math.MaxUint64
				}
			}
		} else {
			// 如果是最后一次调用，返回选择的结果
			if member != nil {
				if strings.HasPrefix(member.Name, roPrefix) {
					return 1
				}
			} else if method != nil {
				return method.getRoResultFlag()
			}
		}
	}
	return 0
}
