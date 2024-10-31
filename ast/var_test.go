package ast

import (
	"readonly/tools"
	"testing"
)

/*
ast var 可能的类型
1. *ast.Field
2. *ast.ValueSpec
3. *ast.AssignStmt
4. *ast.DeclStmt
5. *ast.ForStmt
*/
func TestVar(t *testing.T) {
	tools.PrintTree("var.go", nil)
}
