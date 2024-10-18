package main

type foo struct {
	id int
}

func returnRo() int {
	var ro = 1
	return ro
}

func returnRo2() (a int) {
	var ro = 1
	return ro
}

func returnRo3() foo {
	var ro = foo{id: 1}
	return ro
}

func returnRo4() (foo, foo) {
	var ro = foo{id: 1}
	return foo{}, ro
}

func changeRoParam(roParam int) {
	roParam = 1
}

func readRoParam(roParam int, normalParam int) {
	var roTmp = roParam
	_ = roTmp
}

func basicTypeRoRule() {
	var roV1 = 123
	roV1 = 1 // fail, cannot change value
	var roV2 = roV1
	roV3 := 123
	roV3 = roV2 // fail, cannot change value
	_ = roV3
	var basicVar = roV2 // ok, lhs is basic type
	_ = basicVar
	var roFoo = foo{id: 1}
	o := roFoo
	_ = o
}

func funcRoRule() {
	var roV1 = returnRo() // ok
	var v1 = returnRo()   // fail, cannot receive, must be ro variable
	v1 = returnRo()
	v1 = returnRo2()
	_ = v1
	_ = roV1

	var v2 = returnRo3()
	v2 = returnRo3()
	_ = v2
	var roV3 = returnRo3()
	roV3 = returnRo3()
	_ = roV3
	o, roV4 := returnRo4()
	_, _ = o, roV4

	readRoParam(roV1, v1)   // ok
	readRoParam(v1, v1)     // ok
	readRoParam(roV1, roV1) // normalParam fail, cannot receive ro variable

	changeRoParam(1) // fail, cannot change ro param
}

func main() {
	basicTypeRoRule()
	funcRoRule()
	baseAssign()
}
