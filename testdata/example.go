//golangcitest:args -Ereadonly
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

func ro1() (roInt []int) {
	var arrInt []int
	return arrInt
}
func ro2() (roInt []int, i1 int, i2 []int) {
	var arrInt []int
	return arrInt, 0, nil
}
func ro3(roInt []int, i1 int, i2 []int) {
}

func test() {
	var drain []any

	var roI1, roI2 int = 1, 2
	roI1 = 2        // 不能赋值
	var roI3 = roI2 // 可以初始化

	var roInts1 = ro1()
	var ints1 = ro1() // 不能初始化为非只读
	roInts, i1, roInts3 := ro2()

	ro3(roInts1, i1, nil)
	ro3(roInts1, i1, roInts3) // roInts3 不能用于非只读参数

	drain = append(drain, roI1, roI2, roI3,
		roInts1, ints1, roInts, roInts3,
		i1)
}
