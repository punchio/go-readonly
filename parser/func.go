package parser

//
//import (
//	"go/ast"
//	"log"
//	"slices"
//)
//
//// CollectFuncDecl 获取结构体方法
//func CollectFuncDecl(node ast.Node) {
//	switch d := node.(type) {
//	case *ast.FuncDecl:
//		log.Println("func decl:", d.Name.Name)
//		ast.Inspect(d.Body, func(node ast.Node) bool {
//			expr, ok := node.(*ast.CallExpr)
//			if !ok {
//				return true
//			}
//
//			var sels []string
//			cur := expr.Fun
//			var root *ast.Ident
//		LOOP:
//			for {
//				switch e := cur.(type) {
//				case *ast.CallExpr:
//					cur = e.Fun
//				case *ast.SelectorExpr:
//					cur = e.X
//					sels = append(sels, e.Sel.Name)
//				case *ast.Ident:
//					root = e
//					break LOOP
//				default:
//					panic("unsupported call expr")
//				}
//			}
//			slices.Reverse(sels)
//
//			for i := 0; i < len(sels)-1; i++ {
//				switch e := root.Obj.Decl.(type) {
//				case *ast.FuncDecl:
//					addCallee(d, e)
//					//ident := getFinalIdent(e.Type.Results.List[0].Type)
//					//st := ident.Obj.Decl.(*ast.StructType)
//				case *ast.StructType:
//
//				}
//			}
//			switch e := root.Obj.Decl.(type) {
//			case *ast.FuncDecl:
//				addCallee(d, e)
//			}
//
//			switch e := expr.Fun.(type) {
//			case *ast.Ident:
//				f, ok := e.Obj.Decl.(*ast.FuncDecl)
//				if !ok {
//					return true
//				}
//				log.Println("----call ident:", f.Name.Name)
//				addCallee(d, f)
//			case *ast.SelectorExpr:
//				ast.Inspect(e.X, func(node ast.Node) bool {
//					return true
//				})
//				ident, ok := e.X.(*ast.Ident)
//				if !ok || ident.Obj == nil {
//					return true
//				}
//
//				f, ok := ident.Obj.Decl.(*ast.Field)
//				if !ok {
//					return true
//				}
//
//				se, ok := f.Type.(*ast.StarExpr)
//				if ok {
//					ident = se.X.(*ast.Ident)
//				} else {
//					ident = f.Type.(*ast.Ident)
//				}
//				ts, ok := ident.Obj.Decl.(*ast.TypeSpec)
//				if !ok {
//					return true
//				}
//				info, ok := typeInfos[ts]
//				if !ok {
//					return true
//				}
//				for _, decl := range info.methods {
//					if decl.Name.Name == e.Sel.Name {
//						log.Println("----call selector:", ts.Name.Name, ".", decl.Name.Name)
//						addCallee(d, decl)
//						break
//					}
//				}
//			}
//
//			return true
//		})
//	}
//}
