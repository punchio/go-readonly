package ast

type fooVar struct {
	m  int
	mm map[int]*fooVar
}

type (
	aFoo  fooVar
	myInt int
)

func newFooVar() *fooVar {
	return &fooVar{}
}

func get() int {
	return 1
}

func call(...any) {

}

func (ff fooVar) get() int {
	return 0
}

func (ff fooVar) getMM() map[int]*fooVar {
	return ff.mm
}

func (ff fooVar) getMulti() (int, int) {
	if ff.mm == nil {
		return ff.getMulti()
	}
	return 0, ff.get()
}

func (ff *fooVar) call(pp1 int, pp2 *fooVar, pp3 ...string) (rr1 int, rr2 string) {
	// *ast.Field
	fd0 := ff
	fd1 := ff.m

	// *ast.AssignStmt
	as0 := 1         // *ast.BasicLit
	as1 := fooVar{}  // *ast.CompositeLit
	as2 := pp1       // *ast.Ident
	as3 := &fooVar{} // *ast.StarExpr
	as4 := ff.get()  // *ast.SelectorExpr
	as5 := ff.m      // *ast.SelectorExpr
	as6 := get()     // *ast.CallExpr
	as7 := pp3[0]    // *ast.IndexExpr
	as8 := *pp2      // *ast.StarExpr
	as9 := (*newFooVar()).mm[0].get()
	as10, as11 := (&as1).call(0, nil)
	as12, as13 := call, ff.get
	as12()
	as13()

	// *ast.DeclStmt 与 *ast.AssignStmt差不多，只有不初始化值的区别
	var ds0 fooVar
	var ds1 int
	var ds2 aFoo
	var ds3 = 1
	var ds4 = fooVar{}

	// *ast.RangeStmt
	fs := []int{0, 1}
	for fsi, fsv := range fs {
		fsi++
		fsv++
	}
	for i := 0; i < 10; i++ {
		i++
	}
	fs1 := fooVar{}
	for fsmk, fsmv := range fs1.getMM() {
		fsmk++
		fsmv.get()
	}

	call(fd0, fd1, as0, as1, as2, as3, as4, as5, as6, as7, as8, as9, as10, as11, ds0, ds1, ds2, ds3, ds4, rr1, rr2)
	return
}
