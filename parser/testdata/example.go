package testdata

func (st) get() int { return 0 }

type st struct {
	t st2
}

type st2 struct {
	v int
}

func (st) getSt() st2 { return st2{} }

func (st2) get() int { return 0 }

func r1() int {
	return 0
}

func r2() (int, int) {
	return 0, 0
}
func r3() (int, int, int) {
	return 0, 0, 0
}

func f2() {
	var drain []any
	var vst0 *st
	var vi1, vi2 int

	var v1, v2, v3 = r1(), r1(), r1()
	var v4, v5, v6 = r3()
	var v7, v8, v9 = 1, v5, r1()
	var a = st{}
	var v10 = a.get()
	var v11 = a.t.get()
	var v12 = a.getSt().get()
	var v13 = a.t.v
	v100, v101 := 1, int32(0)

	drain = append(drain, vst0, vi1, vi2,
		v1, v2, v3, v4, v5, v6, v7, v8, v9, v10, v11, v12, v13,
		v100, v101)
}
