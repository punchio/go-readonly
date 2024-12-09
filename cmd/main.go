package main

import (
	"os"
	"readonly"
)

func main() {
	readonly.CheckDir(os.Args[1])
}
