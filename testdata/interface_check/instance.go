package interface_check

func check() {
	s1 := &foo{}
	s2 := &roFoo{}
	receiver(s1)
	receiver(s2)

	receiverRo(s1)
	receiverRo(s2)
}

func receiver(i iGet) {
	ints := i.Get()
	roInts := i.Get()
	_ = ints
	_ = roInts
}

func receiverRo(i iRoGet) {
	ints := i.Get()
	roInts := i.Get()
	_ = ints
	_ = roInts
}
