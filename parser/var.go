package parser

import (
	"fmt"
	"go/ast"
	"go/types"
	"slices"
)

/*
ast var 可能的类型
1. *ast.Field ==> 对应最终类型
2. *ast.AssignStmt ==> 如果左右值数量不等，则对应函数返回值类型；如果相等，右值可能为函数返回值、*ast.Ident、*ast.BasicLit、*ast.CompositeLit
3. *ast.DeclStmt
4. *ast.RangeStmt
*/

/*
ast type 可能的类型
1. expr
	Ident, BasicLit, FuncLit, CompositeLit, ArrayType, MapType, StructType, InterfaceType, FuncType, ChanType
2. stmt
	DeclStmt, AssignStmt
3. decl
	FuncDecl, GenDecl => FuncType, TypeSpec, ValueSpec
4. node // 特殊的
	Field
*/

func getVarType(expr ast.Node) ast.Expr {
	switch t := expr.(type) {
	// 最终类型，所有变量都应该对应到这的某一个类型
	case *ast.BasicLit, *ast.FuncLit, *ast.ArrayType, *ast.MapType, *ast.StructType, *ast.InterfaceType, *ast.FuncType, *ast.ChanType:
		return t.(ast.Expr)

	case *ast.CompositeLit:
		if ident, ok := t.Type.(*ast.Ident); ok {
			return getVarType(ident)
		}
		return t.Type
	case *ast.Field:
		return getVarType(t.Type)
	case *ast.FuncDecl:
		return getVarType(t.Type)
	case *ast.StarExpr:
		return getVarType(t.X)
	case *ast.ParenExpr:
		return getVarType(t.X)
	case *ast.UnaryExpr:
		return getVarType(t.X)
	case *ast.CallExpr:
		return getVarType(t.Fun)
	case *ast.SelectorExpr:
		return getSelectorTypes(t)[0]

	// 标识符，可能为表达式、声明、赋值语句
	case *ast.Ident:
		if t.Obj == nil {
			return t
		}
		switch d := t.Obj.Decl.(type) {
		case *ast.TypeSpec:
			ident, ok := d.Type.(*ast.Ident)
			// 如果是TypeName，则返回name；如果不是，返回type
			if ok && ident.Obj != nil {
				return d.Name
			}
			return getVarType(d.Type)
		case *ast.ValueSpec:
			return getVarType(getVarTypeFromValueSpec(t, d))
		case *ast.Field:
			return getVarType(d)
		case ast.Stmt:
			return getVarType(getVarTypeFromStmt(t, d))
		case ast.Expr:
			return getVarType(d)
		case ast.Decl:
			return getVarType(getVarTypeFromDecl(t, d))
		default:
			panic(fmt.Sprintf("getVarType fail, expr:%s", types.ExprString(t)))
		}
	}
	return nil
}

func getVarTypeFromStmt(ident *ast.Ident, stmt ast.Stmt) ast.Expr {
	switch t := stmt.(type) {
	case *ast.DeclStmt:
		gd := t.Decl.(*ast.GenDecl)
		return getVarTypeFromDecl(ident, gd)
	case *ast.AssignStmt:
		index := 0
		for i, lhs := range t.Lhs {
			tmp := lhs.(*ast.Ident)
			if tmp.Name == ident.Name {
				index = i
				break
			}
		}
		var expr ast.Expr
		if len(t.Lhs) == len(t.Rhs) {
			expr = getVarType(t.Rhs[index])
			index = 0
		} else {
			expr = getVarType(t.Rhs[0])
		}
		switch t := expr.(type) {
		case *ast.FuncType, *ast.FuncLit:
			result := getFuncReturnTypes(t)
			return result[index]
		default:
			return expr
		}
	}
	panic(fmt.Sprintf("getVarTypeFromStmt fail not found, expr:%s", types.ExprString(ident)))
}

func getVarTypeFromValueSpec(ident *ast.Ident, spec *ast.ValueSpec) ast.Expr {
	index := -1
	for i, name := range spec.Names {
		if name.Name == ident.Name {
			index = i
			break
		}
	}

	if index == -1 {
		return nil
	}

	var expr ast.Expr
	if len(spec.Names) == len(spec.Values) {
		expr = getVarType(spec.Values[index])
		index = 0
	} else if spec.Values != nil {
		expr = getVarType(spec.Values[0])
	} else {
		expr = spec.Type
	}

	switch t := expr.(type) {
	case *ast.FuncType, *ast.FuncLit:
		result := getFuncReturnTypes(t)
		return result[index]
	default:
		return expr
	}
}

func getVarTypeFromDecl(ident *ast.Ident, decl ast.Decl) ast.Expr {
	switch t := decl.(type) {
	case *ast.GenDecl:
		// 变量声明的类型不会是TypeSpec
		for _, spec := range t.Specs {
			expr := getVarTypeFromValueSpec(ident, spec.(*ast.ValueSpec))
			if expr == nil {
				continue
			}
			return expr
		}
	case *ast.FuncDecl:
		return t.Type
	}
	panic(fmt.Sprintf("getVarTypeFromDecl fail not found, expr:%s", types.ExprString(ident)))
}

// getFuncReturnTypes 获得函数调用返回值类型
func getFuncReturnTypes(expr ast.Expr) []ast.Expr {
	switch t := expr.(type) {
	case *ast.CallExpr:
		return getFuncReturnTypes(t.Fun)
	case *ast.SelectorExpr:
		return getSelectorTypes(t)
	case *ast.Ident:
		typ := getVarType(t)
		return getFuncReturnTypes(typ)
	case *ast.FuncLit:
		var result []ast.Expr
		for _, field := range t.Type.Results.List {
			result = append(result, field.Type)
		}
		return result
	case *ast.FuncType:
		var result []ast.Expr
		for _, field := range t.Results.List {
			result = append(result, field.Type)
		}
		return result
	default:
		panic(fmt.Sprintf("getFuncReturnTypes fail, expr:%s", types.ExprString(expr)))
	}
}

// getSelectorTypes 选择的过程中，只能出现变量、函数、方法、索引表达式
func getSelectorTypes(sel *ast.SelectorExpr) []ast.Expr {
	ident, names := getSelectorSequence(sel)
	expr := getVarType(ident)
	for i, name := range names {
		switch t := expr.(type) {
		case *ast.BasicLit, *ast.ChanType:
			panic(fmt.Sprintf("getSelectorTypes fail, expr:%s", types.ExprString(expr)))
		case *ast.ArrayType:
			expr = t.Elt
		case *ast.MapType:
			expr = t.Value
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
			}
		case *ast.InterfaceType:
			for _, field := range t.Methods.List {
				if field.Names[0].Name == name {
					expr = field.Type
					break
				}
			}
		case *ast.FuncType, *ast.FuncLit:
			result := getFuncReturnTypes(t)
			if i != len(names)-1 {
				expr = result[0]
			} else {
				return getFuncReturnTypes(t)
			}
		}
	}
	return []ast.Expr{expr}
}

// getSelectorSequence 选择表达式只会出现call,index,selector三种情况，且最终以ident结束
func getSelectorSequence(sel *ast.SelectorExpr) (*ast.Ident, []string) {
	var sels = []string{sel.Sel.Name}
	cur := sel.X
	var root *ast.Ident
LOOP:
	for {
		switch e := cur.(type) {
		case *ast.CallExpr:
			cur = e.Fun
			sels = append(sels, "")
		case *ast.SelectorExpr:
			cur = e.X
			sels = append(sels, e.Sel.Name)
		case *ast.IndexExpr:
			cur = e.X
			sels = append(sels, "")
		case *ast.StarExpr:
			cur = e.X
		case *ast.ParenExpr:
			cur = e.X
		case *ast.UnaryExpr:
			cur = e.X
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
