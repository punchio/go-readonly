package node1

type info struct {
}

func testFuncRoAssign() {
	var roInt = 1
	_ = roInt
	roInt = 2 // 非法，应该报错，不能赋值

	var roInt2 = 2
	roInt2 = roInt
	_ = roInt2 // 非法，应该报错，不能赋值

	roInt3 := roInt // 合法，用别的只读变量初始化
	_ = roInt3
}

func testFuncRoDeclare() {
	var roInt = 1
	_ = roInt

	var roInt2 = roInt // 合法，用别的只读变量初始化
	_ = roInt2
}
