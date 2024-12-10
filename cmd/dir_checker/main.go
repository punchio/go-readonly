package main

import (
	"github.com/punchio/go-readonly"
	"os"
)

func main() {
	readonly.CheckDir(os.Args[1])
}
