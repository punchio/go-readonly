package testdata

func callBefore() m3 {
	m := m1{}
	return m.declAfter().m
}

func newM1() *m1 {
	return &m1{}
}

type m1 struct {
	m m2
}

func (m *m1) declAfter() *m2 {
	return &m.m
}

func (m *m1) declAfter2() *m2 {
	return &m.m
}

func (m *m2) declBefore() m3 {
	return m.m
}

type m2 struct {
	m m3
}

func (m *m2) declAfter() m3 {
	return m.m
}

type m3 struct {
	m int
}

func (m m3) noop() {

}

func callAfter() m3 {
	m := m1{}
	return m.m.declBefore()
}

func call() {
	// normal
	m := newM1()
	//	sel-call-call
	before := m.declAfter().declBefore()
	//	sel-sel-call
	before = m.m.declBefore()
	// call-call-call
	before = newM1().declAfter().declBefore()
	// call-sel-call
	before = newM1().m.declBefore()

	before.noop()
}
