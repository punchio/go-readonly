package readonly

import (
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"go/types"
	"strings"
)

const roPrefix = "ro"

type funcCallTree struct {
	funcSet map[*ast.FuncType]*funcInfo
}

type funcInfo struct {
	mask           int
	callee, caller []*ast.FuncType
}

type structInfo struct {
	fields  []*ast.Field
	methods []*ast.FuncDecl
}

func collectRoFuncSet(funcDecl *ast.FuncDecl, roFuncSet map[*ast.FuncType]int) {
	// 找到函数或方法声明
	// 检查返回值的命名
	if funcDecl.Type.Results != nil {
		index := 0
		for _, field := range funcDecl.Type.Results.List {
			for _, name := range field.Names {
				// 检查返回值是否以 "ro" 开头
				if strings.HasPrefix(name.Name, roPrefix) {
					roFuncSet[funcDecl.Type] |= 1 << index
				}
				index++
			}
		}
	}
	// 遍历函数体的所有语句，查找返回语句
	ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.ReturnStmt:
			for i, result := range node.Results {
				// 如果返回的是标识符（变量）
				if ident, ok := result.(*ast.Ident); ok {
					// 检查标识符是否以 "ro" 开头
					if strings.HasPrefix(ident.Name, roPrefix) {
						roFuncSet[funcDecl.Type] |= 1 << i
					}
				}
			}
		case *ast.CallExpr:

		}
		return true
	})

	if funcDecl.Type.Params != nil {
		index := 32
		for _, field := range funcDecl.Type.Params.List {
			for _, name := range field.Names {
				// 检查返回值是否以 "ro" 开头
				if strings.HasPrefix(name.Name, roPrefix) {
					roFuncSet[funcDecl.Type] |= 1 << index
				}
				index++
			}
		}
	}
}

// isValidAssignment 检查赋值是否有效
func checkExprPrefix(expr ast.Expr, prefix string) bool {
	switch e := expr.(type) {
	case *ast.Ident:
		// 直接的标识符
		return strings.HasPrefix(e.Name, prefix)
	case *ast.SelectorExpr:
		// 结构体字段，检查嵌套情况
		return strings.HasPrefix(e.Sel.Name, prefix)
	default:
		return false
	}
}

func getFinalExpr(expr ast.Expr) ast.Expr {
	switch e := expr.(type) {
	case *ast.SelectorExpr:
		return getFinalExpr(e.X)
	default:
		return e
	}
}

// 判断左值是否为基础类型
func isBasicType(info *types.Info, expr ast.Expr) bool {
	if obj, ok := info.Types[expr]; ok {
		if basic, ok := obj.Type.Underlying().(*types.Basic); ok {
			switch basic.Kind() {
			case types.Bool,
				types.Int, types.Int8, types.Int16, types.Int32, types.Int64,
				types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64, types.Uintptr,
				types.Float32, types.Float64,
				types.Complex64, types.Complex128,
				types.String:
				return true
			default:
				return false
			}
		}
	}
	return false
}

func getExprFuncType(expr ast.Expr, info *types.Info) *ast.FuncType {
	ce, ok := expr.(*ast.CallExpr)
	if !ok {
		return nil
	}

	// 有可能是类型强制转换，不影响只读属性可以忽略
	t := info.TypeOf(ce.Fun)
	if t != nil {
		switch t.Underlying().(type) {
		case *types.Signature:
		default:
			return nil
		}
	}

	switch t := ce.Fun.(type) {
	case *ast.Ident:
		d, ok := t.Obj.Decl.(*ast.FuncDecl)
		if !ok {
			return nil
		}
		return d.Type
	case *ast.SelectorExpr:
		return getExprFuncType(t.X, info)
	default:
		return nil
	}
}

func getSelectorFuncType(expr *ast.SelectorExpr, structInfo map[*ast.StructType]*structInfo) *ast.FuncType {
	var fieldNameStack []string
	var cur ast.Expr = expr
	var ident *ast.Ident
	for ident == nil {
		switch t := cur.(type) {
		case *ast.SelectorExpr:
			fieldNameStack = append(fieldNameStack, t.Sel.Name)
			cur = t.X
		case *ast.CallExpr:
			cur = t.Fun
		case *ast.Ident:
			ident = t
		}
	}

	var curType *ast.StructType
	var obj = ident.Obj
	for i := len(fieldNameStack) - 1; i > 0; i-- {
		switch obj.Kind {
		case ast.Typ:
			ts := obj.Decl.(*ast.TypeSpec)
			curType = ts.Type.(*ast.StructType)
			info, ok := structInfo[curType]
			if !ok {
				return nil
			}
		FIELD:
			for _, v := range info.fields {
				for _, name := range v.Names {
					if name.Name == fieldNameStack[i] {
						obj = name.Obj
						break FIELD
					}
				}
			}
		METHOD:
			for _, v := range info.methods {
				if v.Name.Name == fieldNameStack[i] {
					obj = v.Type.Results.List[0].Names[0].Obj
					break METHOD
				}
			}
		default:
			return nil
		}
	}

	if obj.Kind != ast.Fun {
		return nil
	}

	d, ok := obj.Decl.(*ast.FuncDecl)
	if !ok {
		return nil
	}

	return d.Type
}

func checkValueSpec(vs *ast.ValueSpec, fset *token.FileSet, info *types.Info) error {
	if len(vs.Values) == 0 {
		return nil
	}

	if len(vs.Values) == len(vs.Names) {
		for i, value := range vs.Values {
			identExpr := vs.Names[i]
			// 如果是 ro 变量，则跳过
			if checkExprPrefix(identExpr, roPrefix) {
				continue
			}
			// 否则需要检查赋值是否为只读变量
			if checkExprPrefix(value, roPrefix) {
				// 如果不是基础类型，则报错
				finalExpr := getFinalExpr(value)
				if !isBasicType(info, finalExpr) {
					valueText := getExprText(fset, value)
					return fmt.Errorf("var:[%s] declare with ro value:[%s] at %v\n", identExpr.Name, valueText, fset.Position(identExpr.Pos()))
				}
			}
		}
	}

	return nil
}

func checkCallExpr(ce *ast.CallExpr, info *types.Info, roFuncSet map[*ast.FuncType]int) int {
	ft := getExprFuncType(ce, info)
	mask := 0
	if ft == nil {
		mask = ^0
	} else {
		m, ok := roFuncSet[ft]
		if ok {
			mask = m
		}
	}
	return mask
}

// getExprText 将表达式转为源码文本
func getExprText(fset *token.FileSet, expr ast.Node) string {
	var buf strings.Builder
	err := printer.Fprint(&buf, fset, expr)
	if err != nil {
		return ""
	}
	return buf.String()
}
