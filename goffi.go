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

package libgoffi

/*
#include <ffi.h>
#include <stdlib.h>

typedef void (*_fnptr_t)(void);
typedef void* _pointer;
*/
import "C"

import (
	"errors"
	"github.com/achille-roussel/go-dl"
	"reflect"
	"strings"
	"sync"
	"unsafe"
)

type functionPointer C._fnptr_t

var (
	errNoGoFuncDef       = errors.New("parameter is not a Go function definition")
	errNoCFuncDef        = errors.New("parameter is not a C function definition")
	errGoFuncMultiReturn = errors.New("multiple return values for Go impossible (except error as second return value)")
)

type status int

func (s status) Error() string {
	switch s {
	case ffiOk:
		return "ok"
	case ffiBadTypedef:
		return "bad typedef"
	case ffiBadABI:
		return "bad abi"
	}
	return "unknown"
}

const (
	ffiOk         status = C.FFI_OK
	ffiBadTypedef status = C.FFI_BAD_TYPEDEF
	ffiBadABI     status = C.FFI_BAD_ABI
)

type Mode = dl.Mode

const (
	BindLazy   = dl.Lazy
	BindNow    = dl.Now
	BindLocal  = dl.Local
	BindGlobal = dl.Global
)

type Library struct {
	lib         dl.Library
	m           sync.Mutex
	cifCache    map[string]*C.ffi_cif
	symbolCache map[string]uintptr
}

func NewLibrary(library string, mode Mode) (*Library, error) {
	if !strings.Contains(library, "/") {
		path, err := dl.Find(library)
		if err != nil {
			return nil, err
		}
		library = path
	}

	lib, err := dl.Open(library, mode)
	if err != nil {
		return nil, err
	}

	return &Library{
		lib:         lib,
		cifCache:    make(map[string]*C.ffi_cif, 0),
		symbolCache: make(map[string]uintptr, 0),
	}, nil
}

func (l *Library) Close() error {
	for _, cif := range l.cifCache {
		if cif.arg_types != nil {
			C.free(unsafe.Pointer(cif.arg_types))
		}
	}
	return l.lib.Close()
}

func (l *Library) Symbol(name string) (uintptr, error) {
	l.m.Lock()
	defer l.m.Unlock()
	symbol := l.symbolCache[name]
	if symbol != 0 {
		return symbol, nil
	}

	s, err := l.lib.Symbol(name)
	if err != nil {
		return 0, err
	}

	l.symbolCache[name] = symbol
	return s, nil
}

func (l *Library) Import(symbol string, target interface{}) error {
	tpt := reflect.TypeOf(target)

	if tpt.Kind() != reflect.Ptr || tpt.Elem().Kind() != reflect.Func {
		return errors.New("target not a function type")
	}
	tv := reflect.ValueOf(target)
	tv = reflect.Indirect(tv)
	tt := tv.Type()

	returnsError, err := precheckResultTypes(tt)
	if err != nil {
		return err
	}

	outType := wrapReturnType(tt)
	inTypes, inTypesPtr, nargs, err := wrapArgumentTypes(tt)
	if err != nil {
		return err
	}

	cif, err := l.getOrCreateCif(symbol, outType, inTypesPtr, nargs)
	if err != nil {
		return err
	}

	funcPtr, err := l.makeFunctionPointer(symbol)
	if err != nil {
		return err
	}

	stub := makeStub(tt, tt, cif, funcPtr, outType, inTypes, returnsError)
	funcValue := reflect.MakeFunc(tt, stub)
	tv.Set(funcValue)
	return nil
}

func (l *Library) NewImport(symbol string, retType reflect.Type, returnsError bool,
	argumentTypes ...reflect.Type) (interface{}, error) {

	out := []reflect.Type{retType}
	if returnsError {
		out = append(out, TypeError)
	}

	funcType := reflect.FuncOf(argumentTypes, out, false)
	return l.NewImportComplex(symbol, funcType, funcType)
}

func (l *Library) NewImportComplex(symbol string, goFnType reflect.Type, cFnType reflect.Type) (interface{}, error) {
	if goFnType.Kind() != reflect.Func {
		return nil, errNoGoFuncDef
	}
	if cFnType.Kind() != reflect.Func {
		return nil, errNoCFuncDef
	}

	returnsError, err := precheckResultTypes(goFnType)
	if err != nil {
		return nil, err
	}

	outType := wrapReturnType(cFnType)
	inTypes, inTypesPtr, nargs, err := wrapArgumentTypes(cFnType)
	if err != nil {
		return nil, err
	}

	cif, err := l.getOrCreateCif(symbol, outType, inTypesPtr, nargs)
	if err != nil {
		return nil, err
	}

	funcPtr, err := l.makeFunctionPointer(symbol)
	if err != nil {
		return nil, err
	}
	stub := makeStub(goFnType, cFnType, cif, funcPtr, outType, inTypes, returnsError)
	return reflect.MakeFunc(goFnType, stub).Interface(), nil
}

func (l *Library) getOrCreateCif(symbol string, retType ffiType, inTypesPtr *ffiType, nargs int) (*C.ffi_cif, error) {
	l.m.Lock()
	defer l.m.Unlock()

	cif := l.cifCache[symbol]
	if cif == nil {
		c, err := newCif(retType, inTypesPtr, nargs)
		if err != nil {
			return nil, err
		}

		cif = c
		l.cifCache[symbol] = c
	}

	return cif, nil
}

func (l *Library) makeFunctionPointer(name string) (functionPointer, error) {
	symbol, err := l.Symbol(name)
	if err != nil {
		return nil, err
	}
	return (functionPointer)(unsafe.Pointer(symbol)), nil
}

func newCif(retType ffiType, inTypesPtr *ffiType, nargs int) (*C.ffi_cif, error) {
	var cif C.ffi_cif

	retval := status(C.ffi_prep_cif(&cif, C.FFI_DEFAULT_ABI, C.uint(nargs), retType, inTypesPtr))
	if retval != ffiOk {
		return nil, retval
	}

	return &cif, nil
}

func precheckResultTypes(fnType reflect.Type) (bool, error) {
	returnsError := false
	if fnType.NumOut() > 1 {
		if fnType.NumOut() > 2 {
			return false, errGoFuncMultiReturn
		}
		if !fnType.Out(1).Implements(TypeError) {
			return false, errGoFuncMultiReturn
		}
		returnsError = true
	}
	return returnsError, nil
}

func wrapArgumentTypes(fnType reflect.Type) ([]ffiType, *ffiType, int, error) {
	nargs := fnType.NumIn()

	if nargs == 1 {
		ot := fnType.In(0)
		switch ot {
		case TypeVoid:
			// if only parameter is of type void, just ignore passing parameters at all
			return nil, nil, 0, nil
		}
	}

	arguments := (*ffiType)(C.malloc(C.size_t(ptrSize * nargs)))

	header := reflect.SliceHeader{Data: uintptr(unsafe.Pointer(arguments)), Len: nargs, Cap: nargs}
	in := *(*[]ffiType)(unsafe.Pointer(&header))

	for i := 0; i < fnType.NumIn(); i++ {
		ot := fnType.In(i)
		t := wrapType(ot)
		switch ot {
		case TypeVoid:
			C.free(unsafe.Pointer(arguments))
			return nil, nil, 0, errors.New("void is not a legal parameter type")
		}
		in[i] = t
	}
	return in, arguments, nargs, nil
}

func wrapReturnType(fnType reflect.Type) ffiType {
	retType := typeVoid
	if fnType.NumOut() > 0 {
		retType = wrapType(fnType.Out(0))
	}
	return retType
}
