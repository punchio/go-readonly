package pkg

import "readonly/parser/testdata/pkg2"

func NormalCall(p1, p2 int) {

}

func CallRoParam(roArray []int, p2 int) {

}

func CallRoResult(roArray []int, p2 int) (roMap map[int]int, i int) {
	return
}

func CallRoPropagate() []int {
	return pkg2.CallRoResult()
}
