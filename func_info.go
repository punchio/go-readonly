package readonly

import (
	"go/ast"
	"math"
)

type funcInfo struct {
	fullName string
	decl     *ast.FuncDecl
	roMask   uint64 // bit: 0,1,...31 param;32,...62 result;63 receiver
}

// calcDecl 检查函数声明
func (i *funcInfo) calcDecl() {
	roMask := uint64(0)

	decl := i.decl
	funcType := decl.Type
	// 检查接收器
	if decl.Recv != nil && len(decl.Recv.List) > 0 &&
		len(decl.Recv.List[0].Names) > 0 &&
		checkName(decl.Recv.List[0].Names[0].Name) {
		roMask = 1 << 63
	}

	// 检查参数
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

	i.roMask |= roMask
}

// calcBody 检查函数体内返回语句
func (i *funcInfo) calcBody() {
	roMask := uint64(0)
	decl := i.decl
	funcType := decl.Type
	if decl.Body != nil && funcType.Results != nil {
		// 检测返回语句中的只读变量
		index := 32
		for _, stmt := range decl.Body.List {
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
	i.roMask |= roMask
}

func (i *funcInfo) getResultFlag() uint64 {
	return (i.roMask & (uint64(math.MaxUint32) << 32)) >> 32
}

func (i *funcInfo) getParamFlag() uint64 {
	return i.roMask & math.MaxUint32
}

func (i *funcInfo) getRecvFlag() uint64 {
	return i.roMask & (1 << 63)
}
func (i *funcInfo) getIdent() *ast.Ident {
	ident := ast.NewIdent(i.decl.Name.Name)
	ident.Obj = ast.NewObj(ast.Fun, ident.Name)
	ident.Obj.Decl = i.decl
	return ident
}
func (i *funcInfo) getDecl() *ast.FuncDecl {
	return i.decl
}
