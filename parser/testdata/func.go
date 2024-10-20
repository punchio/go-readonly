package testdata

type foo struct {
	items []int
}

func newFoo() *foo {
	return &foo{}
}

func (f *foo) m1() {
	f.m2()
	f.m3()
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
	return roItems, 1
}

func callF1(f *foo) {
	f.m1()
}

func callF2(f *foo) {
	f.m2()
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
