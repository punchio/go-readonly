package main

func baseAssign() {
	// := 初始化方式
	roInt1 := 1
	roString1 := "hello"
	roInterface1 := any(1)
	_ = roInt1
	_ = roString1
	_ = roInterface1

	// var 声明方式
	var roInt2 = 2
	_ = roInt2

	// 只读变量不能被赋值，这里都报错
	roInt1 = 2
	roInt2 = roInt1
	roInterface1 = nil

	// 只读变量只能被用于初始化只读变量，除非被初始化变量是基础类型，如int，string等

	// 初始化基础类型
	varInt := roInt1
	varString := roString1

	_ = varInt
	_ = varString

	//	初始化非基础类型，报错
	varInterface := roInterface1
	_ = varInterface
}
