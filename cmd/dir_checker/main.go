package main

import (
	"github.com/punchio/go-readonly"
	"os"
)

func main() {
	readonly.Run(os.Args[1:]...)
}
