package parser

import (
	"fmt"
	"testing"
)

func TestVisitor(t *testing.T) {
	dir := "./testdata" // 替换为工程目录路径

	_, files, err := ParseDir(dir)
	if err != nil {
		fmt.Printf("Error parsing directory: %v\n", err)
		return
	}

	ProcessFiles(files)
}
