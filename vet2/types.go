package vet

import (
	"go/ast"
	"math"
)

type funcInfo struct {
	fullName string
	decl     *ast.FuncDecl
	roMask   uint64 // bit: 0,1,...31 param;32,...63 result
}

func (i *funcInfo) calcMask(local bool) {
	if local {
		i.roMask = calcFuncMask(i.decl, nil)
	} else {
		i.roMask = calcFuncMask(i.decl, i.decl.Body)
	}
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
