package node1

type info struct {
}

// testFuncRoAssign 判断 *ast.AssignStmt 的tok，来区分是赋值token.ASSIGN，还是初始化token.DEFINE
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

// testFuncRoDeclare 判断 *ast.DeclStmt 声明，是否符合规范 GenDecl
func testFuncRoDeclare() {
	var roInfo = info{}
	_ = roInfo

	var roInfo2 = roInfo // 合法，用别的只读变量初始化
	_ = roInfo2

	var i = roInfo // 非法，只读变量不能初始化到普通变量上
	_ = i
}
