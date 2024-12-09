//golangcitest:args -Ereadonly
package testdata

const (
	a, b = 1, ""
)

type MyType int
type Wrapper struct {
	Inner MyType
}
type foo struct {
	items []int
}

type bar struct {
}

func (bar) noop() {
	var (
		vi, vj   int32 = 1, 2
		vii, vjj int32 = vi, vj
		vs, vt         = "", ""
	)
	var wp Wrapper
	//vs = fmt.Sprintln(vi, vj)
	wp.Inner = MyType(vi)
	_, _, _, _ = vi, vj, vs, vt
	_, _ = vii, vjj
	var i, j = 1, ""
	var ii, ij = 1, 2
	_, _ = i, j
	_, _ = ii, ij
	s := ""
	_ = s
	ia := []int{1}
	_ = ia
	var ia2 []int
	_ = ia2

	var ff, roFf = foo{}, foo{}
	ff.m3()[1] = 0

	callRoParam(&ff, &ff)
	callRoParam(&roFf, &roFf)
	callRoParam(&ff, &roFf)
	callRoParam(&roFf, &ff)

	b := bar{}
	b.noop()

	f := foo{}
	f.m3()[1] = 0
	f.getItems2()[0] = 0

	var roMap map[int]int = map[int]int{}
	roMap[0] = 1

	var roSlice []int = f.getItems2()
	roSubSlice := roSlice[0:1]
	_ = roSubSlice
	subSlice := roSubSlice
	_ = subSlice
	var sliceItem = roSlice[0]
	_ = sliceItem
	for i, roV := range roSlice {
		i = 0
		roV = 1
		_, _ = i, roV
	}
	for i, v := range roSlice {
		i = 0
		v = 1
		_, _ = i, v
	}
	var sliceSt = []foo{}
	for _, roV := range sliceSt {
		_ = roV
	}
	var roSliceSt = []foo{}
	for _, roV := range roSliceSt {
		_ = roV
	}
	for _, v := range roSliceSt {
		_ = v
	}
	var mapSt map[int]foo
	for k, roV := range mapSt {
		_, _ = k, roV
	}
	var roSliceMap = map[int]foo{}
	for k, v := range roSliceMap {
		_, _ = k, v
	}
	for k, roV := range roSliceMap {
		_, _ = k, roV
	}
	var roStStMap = map[*foo]foo{}
	for k, v := range roStStMap {
		_, _ = k, v
	}
	for k, roV := range roStStMap {
		_, _ = k, roV
	}
	for roK, roV := range roStStMap {
		_, _ = roK, roV
	}
}

func newFoo() *foo {
	return &foo{}
}

func (f *foo) m1(b bar, sb *bar) {
	f.m2()
	f.m3()
	b.noop()
	sb.noop()
	i := bar{}
	i.noop()
	ip := &bar{}
	ip.noop()
}

func (f *foo) m2() {
	f.m3()
}

func (f *foo) m3() []int {
	callF1(f)
	return f.getItems()
}

func (f *foo) getItems() []int {
	roItems := f.items
	return roItems
}

func (f *foo) getItems2() (roItems []int) {
	roItems = f.items
	return
}

func (f *foo) getItems3() (roItems []int) {
	another := f.items
	return another
}

func (f *foo) get() int {
	return 1
}

func (f *foo) get2() ([]int, int) {
	roItems := f.items
	roInt := 1
	return roItems, roInt
}

func callF1(f *foo) {
	f.m1(bar{}, &bar{})
}

func getManySame() (int, int, int) {
	return 0, 0, 0
}

func getManyNotSame() (int, uint, string) {
	return 0, 0, ""
}

func callF2(f *foo) {
	var same1, same2, same3 int = getManySame()
	var notSame1, notSame2, notSame3 = getManyNotSame()
	_, _, _ = same1, same2, same3
	_, _, _ = notSame1, notSame2, notSame3
	f.m2()
	var i = f.getItems2()
	var roI = f.getItems()
	var i2, iInt = f.get2()
	roI2, iInt := f.get2()
	var i3 = getRoItems3(f)
	roI3 := getRoItems3(f)
	_ = roI3
	_ = i3
	_ = i
	_ = roI
	_ = i2
	_ = iInt
	_ = roI2
	//pkg.CallNormal(1, 2)
	//var roArr, arr []int
	//var roInt int
	//pkg.CallRoParam(roArr, 1)
	//pkg.CallRoParam(arr, roInt)
	//roMap, _ := pkg.CallRoResult(roArr, roInt)
	//_ = roMap
	//var roPropagate, propagate = pkg.CallRoPropagate(), pkg.CallRoPropagate()
	//_, _ = roPropagate, propagate
}

func callCall(f *foo) foo {
	callF1(f)
	return *f
}

func getRoItems(f *foo) []int {
	var items = f.getItems2()
	return items
}

func getRoItems2(f *foo) []int {
	items := f.getItems2()
	return items
}

func getRoItems3(f *foo) []int {
	return f.getItems2()
}

func callRoParam(f *foo, roF *foo) {
	var i = getRoItems(f)
	_ = i
}
