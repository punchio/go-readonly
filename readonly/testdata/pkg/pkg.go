package pkg

import (
	"fmt"
	"readonly/parser/testdata/pkg2"
)

type st struct {
}

func (i *st) Call([]int) []int {
	return nil
}

func CallNormal(p1, p2 int) {

}

func CallRoParam(roArray []int, p2 int) {

}

func CallRoResult(roArray []int, p2 int) (roMap map[int]int, i int) {
	return
}

func CallRoPropagate() []int {
	fmt.Println("ppppp")
	//a := st{}
	//i := a.Call(pkg2.CallRoResult(1))
	bar := pkg2.GetBar()
	bar.Get()
	bar2 := pkg2.Bar{}
	bar2.Get()
	return pkg2.CallRoResult(1)
}

func returnRo() []int {
	return returnRo2()
}

func returnRo2() []int {
	return returnRo()
}
