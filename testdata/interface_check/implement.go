package interface_check

type foo struct {
}

func (f *foo) Get() []int {
	return nil
}

type roFoo struct {
}

func (r *roFoo) Get() (ro []int) {
	return nil
}
