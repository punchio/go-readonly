# go-readonly
代码中，有的变量不希望被修改，需要赋予只读属性，但是go语言没有只读类型，需要通过添加新的机制来实现这个功能。
目前考虑只有变量有只读需求，所以只实现变量的只读属性  
  注：如果方法的接收器变量是只读变量，则这个方法就是只读方法  
只读变量有以下几个特性：
1. 只读变量不能被修改，只能在初始化时赋值
2. 当变量作为返回值时，接收变量也必须是只读变量
3. 传入的变量是只读变量时，函数的接收参数也必须为只读变量
为了实现只读变量的检查，通过分析语法树，对赋值、初始化、函数返回值、函数入参做相应的检查
4. 只读变量只能调用只读方法，不能调用非只读方法

## 变量赋值、初始化检查
变量初始化、赋值两种情况的检查
- 基础类型
1. 如果变量的类型为基础类型，不用检查其对应的只读类型
2. _ 变量不用检查，可能用在多值返回时，不需要只读变量的情况
- 数量
1. 多左值，多右值
必然左值和右值数量相等，且右值不能有多返回值函数
2. 多左值，单右值
必然是右值为多返回值函数或者方法
- 左值变量类型区别
1. 赋值可以为结构体变量的字段赋值，初始化声明只能是新增变量
- 只读限制区别
1. 只读字段不能再被赋值，所以赋值语句中左值中有只读变量都是错误；所以右值也不能有只读变量
2. 初始化时，右值为只读时，左值也需要为只读变量；右值不为只读，左值没限制；所以，只用检查右值只读变量对应的左值即可
- 需要检查的语句
1. ast.AssignStmt 赋值语句，要区分对待 := 初始化语句和 = 赋值语句
2. ast.DeclStmt 声明语句

## 函数检查
函数需要检查返回值和入参
- 返回值
1. 需要记录只读返回值对应的索引，如多返回值中第几个值为只读
2. 函数调用接收者中，只需要对返回值中只读的索引对应的接收者做只读检查
- 入参
1. 函数参数中，通用需要记录对应参数第几个为只读
2. 函数调用时，需要对调用传入参数为只读变量的索引做检查，是否与函数参数声明中只读参数匹配








# 需要关注的节点类型
stmt、decl、expr三类节点中，根据stmt做检查，decl提供类型信息
- stmt 节点  
  需要关注 SendStmt、IncDecStmt、AssignStmt、ExprStmt
- expr 节点  
  需要关注 Ident、Ellipsis（只读切片可以在可变参数使用）、FuncLit、CompositeLit、
  SelectorExpr、IndexExpr、IndexListExpr、SliceExpr、CallExpr、StarExpr、UnaryExpr、KeyValueExpr

# 规则
## 定义
- 只读类型：以roXxx开头的变量 或者 以roXxx开头的结构体字段
```go
package ro

type info struct {
  
}

type conf struct {
  roInfo *info 
}

func init() {
  var roInt = 1
  var roInfo = &info{}
  var c = &conf{roInfo: roInfo}
  _, _, _ = roInt, roInfo, c
}

```
- 只读方法：结构体方法以RoXxx()开头，则不能修改结构体的数据
```go
package ro

type conf struct {
  data []int
}

func (c *conf) RoCheck(data []int)  {
  c.data = data // 非法
  for i, v := range data {
    c.data[i] = v // 非法
  }

  for i, v := range c.data {
    data[i] = v // 合法
  }
}
```
## 检查情况

- 声明语句  
  使用 *ast.GenDecl 和 *ast.ValueSpec 判断变量声明及初始化
  - 声明语句中，如果右值有只读变量，则左值只能是只读类型
  
- 赋值语句  
  使用 *ast.AssignStmt 中的 Tok 字段判断是否为 := ，如果 Tok 是 = ，表示是赋值操作；如果 Tok 是 := ，表示是声明
  - 如果是赋值操作，且左值是只读变量，则报错
  - 如果是声明，则规则同***声明语句***

- 函数或者方法调用
    1. 如果函数入参为只读类型，则函数对应的参数名也必须是ro开头
    2. 如果函数返回值是只读类型，则对应接收的返回值也必须是ro开头

- 通道
    1. 只读数据放入通道，则通道也需要是只读的

- 取址
    1. 对只读类型取地址，也要符合***赋值表达式***规则

## 检查规则
1. 只读变量只能在初始化时赋值，其他时候都不能赋值
```go
package ro

func init() {
	var roInt = 1 // 合法
	roInt = 2 // 非法
	var roInt2 = roInt // 合法
	roInt2 = 3 // 非法
	roInt3 := roInt2 // 合法
	roInt3 = roInt // 非法
	roInt4 := 4 // 合法
	roInt4 = 1 // 非法
	_, _, _, _ = roInt, roInt2, roInt3, roInt4
}
```
2. 通过只读变量获取的值，如果不是***基础类型***，都必须是只读类型
```go
package ro

type fooStruct struct {
	data *int
}

func roFunc() *fooStruct {
	roFoo := &fooStruct{}
	return roFoo
}
func normalFunc() *fooStruct {
    foo := &fooStruct{}
    return foo
}

func init() {
    roFoo := roFunc() // 合法
	roFoo2 := normalFunc()  // 合法
	foo := roFunc() // 非法，接收变量需要以ro开头
	foo2 := normalFunc()  // 合法
	data := roFoo.data // 非法
	roData := roFoo.data // 合法
	roFoo.data = nil // 非法，不能修改只读变量内容
	_, _, _, _, _, _ = roFoo,roFoo2,foo,foo2,data,roData
}
```

## 检查逻辑
1. 变量在函数内部设为只读时，需要满足***只读规则***
2. 函数入参包含***只读类型***时，需要检查对应函数声明的参数是否为只读
3. 函数返回值包含***只读类型***时，需要检查***接收变量***是不是***只读类型***

# FAQ
- 如何从只读数据里取值，如slice，struct，map等
- struct只读，field可读写，是否还受只读限制
    1. type foo struct { roInt int}
- slice切片考虑这几种情况：
    1. roSlice[i]=1 // 不允许，切片不能改变元素的值
    2. roSubSlice := roSlice[:1] // 允许，只读性质没有变
    3. subSlice := roSlice[:1] // 不允许，只读性质变了，可以被修改了
    4. val := roSlice[0] // 允许，切片只读，但是元素不限制只读
    5. roSlice = append(roSlice, 1) // 不允许，不符合函数规则
    6. s = append(s, roSlice...) // 允许，...之后会生成一个新的切片

## 第一步
1. 支持变量只读检查
2. 支持函数参数只读检查
3. 支持函数返回值只读传递检查