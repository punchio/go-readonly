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

		log.Printf("[rhs:%s at %v ] cannot be assigned to \n\t\t[lhs:%s at %v ] variable without readonly\n",
			rhsBuf.String(), fset.Position(pos), lhsBuf.String(), fset.Position(lhs[i].Pos()))
	}
}

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
			if t.Name == "_" {
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

func collectRhsFlag(rhs []ast.Expr, fset *token.FileSet, info *types.Info) uint64 {
	flag := uint64(0)
	for i, expr := range rhs {
		if t, ok := info.Types[expr]; ok {
			if _, ok = t.Type.(*types.Basic); ok {
				continue
			}
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
				lhsFlag, skipFlag := collectLhsFlag(lhs, fset)
				rhsFlag := collectRhsFlag(rhs, fset, info)
				check(assign, lhs, rhs, lhsFlag, rhsFlag, skipFlag, fset)
			case *ast.DeclStmt:
				decl := e.Decl.(*ast.GenDecl)
				for _, spec := range decl.Specs {
					valueSpec := spec.(*ast.ValueSpec)
					assign, lhs, rhs := collectValueSpec(valueSpec)
					lhsFlag, skipFlag := collectLhsFlag(lhs, fset)
					rhsFlag := collectRhsFlag(rhs, fset, info)
					check(assign, lhs, rhs, lhsFlag, rhsFlag, skipFlag, fset)
				}

				//checkDeclStmt(decl, fset, info)
				//default:
				//	fmt.Printf("stmt:%T, stmt:%v", e, e)
			}
		}
	}
	return nil
}

func checkDeclStmt(decl *ast.GenDecl, fset *token.FileSet, info *types.Info) {
	for _, spec := range decl.Specs {
		vs := spec.(*ast.ValueSpec)
		basicType := false
		skipIndexes := make([]bool, len(vs.Names))
		// 基础类型变量不用检查只读
		if t, ok := info.Types[vs.Type]; ok {
			if _, ok = t.Type.(*types.Basic); ok {
				basicType = true
			}
		}

		// 如果数量不等，必然是一个函数多值返回值;否则，右值都是单一返回值函数或者变量
		if len(vs.Names) != len(vs.Values) {
			// 只读左值不能初始化不用检查
			for i, v := range vs.Names {
				if basicType || strings.HasPrefix(v.Name, roPrefix) || v.Name == "_" {
					skipIndexes[i] = true
				}
			}

			call := vs.Values[0].(*ast.CallExpr)
			flag := getRoFuncFlag(call)
			for i := 0; i < len(vs.Names); i++ {
				if skipIndexes[i] {
					continue
				}
				if flag&(1<<i) != 0 {
					var lhsBuf bytes.Buffer
					_ = printer.Fprint(&lhsBuf, fset, vs.Names[i])
					var rhsBuf bytes.Buffer
					_ = printer.Fprint(&rhsBuf, fset, vs.Values[0])
					log.Printf("Invalid assignment to %s from %s at %v\n", lhsBuf.String(), rhsBuf.String(), fset.Position(vs.Names[i].Pos()))
					return
				}
			}
		} else {
			// 只读左值不能初始化不用检查
			for i, v := range vs.Names {
				if basicType || strings.HasPrefix(v.Name, roPrefix) || v.Name == "_" {
					skipIndexes[i] = true
				}
			}

			for i, value := range vs.Values {
				if skipIndexes[i] {
					continue
				}
				checkFail := false
				switch e := value.(type) {
				case *ast.Ident:
					// 直接的标识符
					checkFail = strings.HasPrefix(e.Name, roPrefix)
				case *ast.SelectorExpr:
					checkFail = strings.HasPrefix(e.Sel.Name, roPrefix)
					// 结构体变量，可能会调用函数，所以要检查选择器flag
					if !checkFail && getSelectorRoFlag(e) > 0 {
						checkFail = true
					}
				case *ast.CallExpr:
					// 检查函数返回值，肯定是一个返回值，所以直接判断是否大于0即可
					if getRoFuncFlag(e) > 0 {
						checkFail = true
					}
				default:
					continue
				}
				if checkFail {
					var lhsBuf bytes.Buffer
					_ = printer.Fprint(&lhsBuf, fset, vs.Names[i])
					var rhsBuf bytes.Buffer
					_ = printer.Fprint(&rhsBuf, fset, value)
					log.Printf("Invalid assignment to %s from %s at %v\n", lhsBuf.String(), rhsBuf.String(), fset.Position(vs.Names[i].Pos()))
					return
				}
			}
		}

	}
}

func checkDefine(assignStmt *ast.AssignStmt, fset *token.FileSet, info *types.Info) {
	skipIndexes := make([]bool, len(assignStmt.Lhs))
	// 只读左值不能被赋值
	for i, lhs := range assignStmt.Lhs {
		// 基础类型变量不用检查只读
		if t, ok := info.Types[lhs]; ok {
			if _, ok = t.Type.(*types.Basic); ok {
				skipIndexes[i] = true
				continue
			}
		}

		ident := lhs.(*ast.Ident)
		if strings.HasPrefix(ident.Name, roPrefix) || ident.Name == "_" {
			skipIndexes[i] = true
		}
	}

	// 右值不能赋值非基础类型
	// 目前只检测了lhs数量和rhs数量一样的情况
	for i, rhs := range assignStmt.Rhs {
		if skipIndexes[i] {
			continue
		}

		checkFail := false
		switch e := rhs.(type) {
		case *ast.Ident:
			// 直接的标识符
			checkFail = strings.HasPrefix(e.Name, roPrefix)
		case *ast.SelectorExpr:
			checkFail = strings.HasPrefix(e.Sel.Name, roPrefix)
			// 结构体字段
			if !checkFail && getSelectorRoFlag(e) > 0 {
				checkFail = true
			}
		case *ast.CallExpr:
			if getRoFuncFlag(e) > 0 {
				checkFail = true
			}
		default:
			continue
		}
		if checkFail {
			var lhsBuf bytes.Buffer
			_ = printer.Fprint(&lhsBuf, fset, assignStmt.Lhs[i])
			var rhsBuf bytes.Buffer
			_ = printer.Fprint(&rhsBuf, fset, rhs)
			log.Printf("Invalid assignment to %s from %s at %v\n", lhsBuf.String(), rhsBuf.String(), fset.Position(assignStmt.Lhs[i].Pos()))
			return
		}
	}
}

func checkAssign(assignStmt *ast.AssignStmt, fset *token.FileSet, info *types.Info) {
	skipIndexes := make([]bool, len(assignStmt.Lhs))
	// 只读左值不能被赋值
	for i, lhs := range assignStmt.Lhs {
		// 基础类型变量不用检查只读
		if t, ok := info.Types[lhs]; ok {
			if _, ok = t.Type.(*types.Basic); ok {
				skipIndexes[i] = true
				continue
			}
		}

		lhsExprName := ""
		switch e := lhs.(type) {
		case *ast.Ident:
			lhsExprName = e.Name
			// 直接的标识符
		case *ast.SelectorExpr:
			// 结构体字段
			ident, _ := getSelectorTree(e)
			lhsExprName = ident.Name
		default:
			panic("unsupported assign lhs")
		}

		if lhsExprName == "_" {
			skipIndexes[i] = true
			continue
		}

		if strings.HasPrefix(lhsExprName, roPrefix) {
			skipIndexes[i] = true
			var buf bytes.Buffer
			_ = printer.Fprint(&buf, fset, lhs)
			log.Printf("Invalid assignment to %s from %s at %v\n", lhsExprName, buf.String(), fset.Position(lhs.Pos()))
			continue
		}
	}

	// 右值不能赋值非基础类型
	// 目前只检测了lhs数量和rhs数量一样的情况
	for i, rhs := range assignStmt.Rhs {
		if skipIndexes[i] {
			continue
		}

		getExprReadonlyFlag(rhs)
		checkFail := false
		switch e := rhs.(type) {
		case *ast.Ident:
			// 直接的标识符
			checkFail = strings.HasPrefix(e.Name, roPrefix)
		case *ast.SelectorExpr:
			checkFail = strings.HasPrefix(e.Sel.Name, roPrefix)
			// 结构体字段
			if !checkFail {
				flag := getSelectorRoFlag(e)
				// 不为0，表示对应结果是只读的
				if flag&(1<<i) != 0 {
					checkFail = true
				}
			}
		case *ast.CallExpr:
			if getRoFuncFlag(e) != 0 {
				checkFail = true
			}
		default:
			continue
		}
		if checkFail {
			var lhsBuf bytes.Buffer
			_ = printer.Fprint(&lhsBuf, fset, assignStmt.Lhs[i])
			var rhsBuf bytes.Buffer
			_ = printer.Fprint(&rhsBuf, fset, rhs)
			log.Printf("Invalid assignment to %s from %s at %v\n", lhsBuf.String(), rhsBuf.String(), fset.Position(assignStmt.Lhs[i].Pos()))
			return
		}
	}
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
		}
	case *ast.CallExpr:
		return getExprDeclType(t.Fun)
	case *ast.SelectorExpr:
		return getExprDeclType(t.X)
	case *ast.StarExpr:
		return getExprDeclType(t.X)
	case *ast.CompositeLit:
		return getExprDeclType(t.Type)
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

func getSelectorRoFlag(sel *ast.SelectorExpr) uint64 {
	// 获取选择器的第一个标识符
	ident, sels := getSelectorTree(sel)
	switch t := ident.Obj.Decl.(type) {
	case *ast.AssignStmt:
		for i, lhs := range t.Lhs {
			tmp := lhs.(*ast.Ident)
			if tmp.Name == ident.Name {
				ts := getExprDeclType(t.Rhs[i])
				ident = ts[0].Name
				break
			}
		}
	case *ast.Field:
		ident = getFinalIdent(t.Type)
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
