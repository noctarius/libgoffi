/*
 * libgoffi - libffi adapter library for Go
 * Copyright 2019 clevabit GmbH
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"fmt"
	goffi "github.com/clevabit/libgoffi"
	"reflect"
)

type getpid = func() (int, error)
type abs = func(int) (int, error)
type fnSqrt = func(float64) float64

func main() {
	println("loading library...")
	lib, err := goffi.NewLibrary("libc", goffi.BindNow|goffi.BindGlobal)
	if err != nil {
		panic(err)
	}

	println("searching getpid function...")
	fn, err := lib.NewImport("getpid", goffi.TypeInt, true)
	if err != nil {
		panic(err)
	}

	println("executing getpid...")
	fnGetpid := fn.(getpid)
	pid, err := fnGetpid()
	if err != nil {
		panic(err)
	}
	println(fmt.Sprintf("pid: %d", pid))

	println("searching abs function...")
	fn, err = lib.NewImport("abs", goffi.TypeInt, true, goffi.TypeInt)
	if err != nil {
		panic(err)
	}

	println("executing abs...")
	fnAbs := fn.(abs)
	a, err := fnAbs(-12)
	//a, err := lib.Test(-12)
	if err != nil {
		panic(err)
	}
	println(fmt.Sprintf("abs: %d", a))

	var sqrt fnSqrt
	err = lib.Import("sqrt", &sqrt)
	if err != nil {
		panic(err)
	}

	println(fmt.Sprintf("sqrt: %f", sqrt(9.)))

	fnGo := reflect.FuncOf(
		[]reflect.Type{goffi.TypeInt},
		[]reflect.Type{goffi.TypeInt},
		false,
	)

	fnC := reflect.FuncOf(
		[]reflect.Type{goffi.TypeFloat64},
		[]reflect.Type{goffi.TypeFloat64},
		false,
	)

	fn, err = lib.NewImportComplex("sqrt", fnGo, fnC)
	if err != nil {
		panic(err)
	}

	sqrt2, ok := fn.(func(int) int)
	if !ok {
		panic("could not map :(")
	}

	println(fmt.Sprintf("sqrt: %d", sqrt2(9)))
}
