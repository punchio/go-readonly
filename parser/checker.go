package parser

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"go/types"
	"log"
	"slices"
	"strings"
)

const roPrefix = "ro"

func checkReadonly(file *ast.File, fset *token.FileSet, info *types.Info) error {
	for _, v := range funcInfos {
		for _, stmt := range v.decl.Body.List {
			_ = stmt
			switch e := stmt.(type) {
			case *ast.AssignStmt:
				if e.Tok == token.ASSIGN {
					checkAssign(e, fset, info)
				} else if e.Tok == token.DEFINE {
					//checkAssignDeclare(e, fset, info, roFuncSet)
				}
			case *ast.DeclStmt:

			}
		}
	}
	return nil
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

		checkFail := false
		switch e := rhs.(type) {
		case *ast.Ident:
			// 直接的标识符
			checkFail = strings.HasPrefix(e.Name, roPrefix)
		case *ast.SelectorExpr:
			checkFail = strings.HasPrefix(e.Sel.Name, roPrefix)
			// 结构体字段
			if !checkFail && isRoSelector(e) {
				checkFail = true
			}
		case *ast.CallExpr:
			if isRoFunc(e, 0) {
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

//func checkAssignDeclare(assignStmt *ast.AssignStmt, fset *token.FileSet, info *types.Info, roFuncs map[*ast.FuncType]int) {
//	needCheck := map[int]bool{}
//	for i, rhs := range assignStmt.Rhs {
//		switch e := rhs.(type) {
//		case *ast.CallExpr:
//			ft := getExprFuncType(e, info)
//			if mask, ok := roFuncs[ft]; !ok || mask&1<<i == 0 {
//				needCheck[i] = true
//			}
//		case *ast.Ident:
//			if strings.HasPrefix(e.Name, roPrefix) && !isBasicType(info, e) {
//				needCheck[i] = true
//			}
//		case *ast.SelectorExpr:
//			if strings.HasPrefix(e.Sel.Name, roPrefix) {
//				needCheck[i] = true
//			}
//		}
//	}
//
//	for i, lhs := range assignStmt.Lhs {
//		if !needCheck[i] {
//			continue
//		}
//		lhsExprName := ""
//		switch e := lhs.(type) {
//		case *ast.Ident:
//			// 直接的标识符
//			lhsExprName = e.Name
//		}
//		if !strings.HasPrefix(lhsExprName, roPrefix) {
//			var buf bytes.Buffer
//			_ = printer.Fprint(&buf, fset, lhs)
//			log.Printf("Invalid declare to %s from %s at %v\n", lhsExprName, buf.String(), fset.Position(lhs.Pos()))
//			return
//		}
//	}
//}

func getSelectorTree(sel *ast.SelectorExpr) (*ast.Ident, []string) {
	var sels []string
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

func isRoFunc(call *ast.CallExpr, index int) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if ok {
		return isRoSelector(sel)
	}
	ident := call.Fun.(*ast.Ident)
	decl := ident.Obj.Decl.(*ast.FuncDecl)
	info := funcInfos[decl]
	return info.isResultRo(index)
}

func getExprDeclType(expr ast.Expr) []*ast.TypeSpec {
	return nil
}

func isRoSelector(sel *ast.SelectorExpr) bool {
	ident, sels := getSelectorTree(sel)
	switch t := ident.Obj.Decl.(type) {
	case *ast.AssignStmt:
		for i, lhs := range t.Lhs {
			tmp := lhs.(*ast.Ident)
			if tmp.Name == tmp.Name {
				ident = t.Rhs[i].(*ast.Ident)
				break
			}
		}
	}
	// 最后一次选择函数可能返回多个值
	cur := ident
	for i := 0; i < len(sels)-1; i++ {
		switch cur.Obj.Kind {
		case ast.Typ:
			decl := cur.Obj.Decl.(*ast.GenDecl)
			typeSpec := decl.Specs[0].(*ast.TypeSpec)
			info := typeInfos[typeSpec]
			member := info.getMember(sels[i])
			if member == nil {
				panic(fmt.Sprintf("struct member not found, member:%s,struct:%v", sels[i], cur.Obj))
			}
			cur = member
		case ast.Fun:
			decl := cur.Obj.Decl.(*ast.FuncDecl)
			cur = decl.Type.Results.List[0].Names[0]
		default:
			panic("unsupported selector")
		}
		if strings.HasPrefix(cur.Name, roPrefix) {
			return true
		}
	}
	return false
}

func getIdentDecl(ident *ast.Ident) ast.Decl {
	switch ident.Obj.Kind {
	case ast.Typ, ast.Fun:
		return ident.Obj.Decl.(ast.Decl)
	case ast.Var:
		_ = ident.Obj.Decl
		return nil
	default:
		return nil
	}
}

func getSelectField(decl ast.Decl, field string) ast.Decl {
	return nil
}
