package main

import (
	"fmt"
	goffi "github.com/clevabit/libgoffi"
)

type getpid = func() (int, error)
type abs = func(int) (int, error)

func main() {
	println("loading library...")
	lib, err := goffi.NewLibrary("libc", goffi.BindNow|goffi.BindGlobal)
	if err != nil {
		panic(err)
	}

	/*println("searching getpid function...")
	fn, err := lib.Function("getpid", goffi.TypeInt, true)
	if err != nil {
		panic(err)
	}

	println("executing getpid...")
	fnGetpid := fn.(getpid)
	pid, err := fnGetpid()
	if err != nil {
		panic(err)
	}
	println(fmt.Sprintf("pid: %d", pid))*/

	println("searching abs function...")
	fn, err := lib.Function("abs", goffi.TypeInt, true, goffi.TypeInt)
	if err != nil {
		panic(err)
	}

	println("executing abs...")
	fnAbs := fn.(abs)
	a, err := fnAbs(-12)
	if err != nil {
		panic(err)
	}
	println(fmt.Sprintf("abs: %d", a))
}
