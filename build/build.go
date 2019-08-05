package main

import (
	"fmt"
	"os"
	"runtime"
)

func main() {
	for _, arg := range os.Args {
		if arg == "-a" {
			fmt.Println(runtime.GOARCH)
		} else if arg == "-o" {
			fmt.Println(runtime.GOOS)
		}
	}
}
