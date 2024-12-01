package pkg

import (
	"fmt"
	"readonly/parser/testdata/pkg2"
)

func CallNormal(p1, p2 int) {

}

func CallRoParam(roArray []int, p2 int) {

}

func CallRoResult(roArray []int, p2 int) (roMap map[int]int, i int) {
	return
}

func CallRoPropagate() []int {
	fmt.Println("ppppp")
	return pkg2.CallRoResult()
}
