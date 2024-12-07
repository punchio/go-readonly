package pkg2

import . "fmt"

type Bar struct {
}

func (b *Bar) Get() int {
	var a int = 2
	c := 1
	Println(a, c)
	return c
}

func GetBar() *Bar {
	return &Bar{}
}

func get([]int) int {
	iii := 1
	return iii
}

func CallRoResult(p int) []int {
	var i int
	roArray := []int{1, 2}
	//var a any = i
	//switch t := a.(type) {
	//default:
	//	Println(t)
	//}
	i = i + 1
	Println(get(nil))
	return roArray
}
