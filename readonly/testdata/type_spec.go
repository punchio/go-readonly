//golangcitest:args -Ereadonly
//golangcitest:config_path testdata/readonly.yml
package testdata

type m1 struct {
	m m2
}

var m m1

func callBefore() m3 {
	m := m1{}
	return m.declAfter().m
}

func newM1() *m1 {
	return &m1{}
}

func (m *m1) declAfter() *m2 {
	return &m.m
}

func (m *m1) declAfter2() *m2 {
	return &m.m
}

func (m *m2) declBefore() m3 {
	roM := m.m
	return roM
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

func (m3) noReceiver() {

}

func (m m3) noop() {

}

func (m3) getM2() m2 {
	return m2{}
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
	newM1().m.declBefore().getM2().m.getM2()

	roMm := newM1()
	roMm.declAfter2()
	roMm = newM1()

	var m2m m2
	m2m = newM1().m.declBefore().getM2().m.getM2()
	m2m.declBefore()

	before.noop()
}

func getExprReturnType() {

}
