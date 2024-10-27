package testdata

type foo struct {
	items []int
}

type bar struct {
}

func (bar) noop() {
	var (
		vi, vj int32 = 1, 2
		vs, vt       = "", ""
	)
	_, _, _, _ = vi, vj, vs, vt
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

	b := bar{}
	b.noop()

	f := foo{}
	f.m3()[1] = 0
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
	_ = i
	_ = roI
	_ = i2
	_ = iInt
	_ = roI2
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
