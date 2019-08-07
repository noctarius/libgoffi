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
	"sync"
	"unsafe"
)

type functionPointer C._fnptr_t

var (
	errNoGoFuncDef              = errors.New("parameter is not a Go function definition")
	errNoCFuncDef               = errors.New("parameter is not a C function definition")
	errGoFuncMultiReturn        = errors.New("multiple return values for Go impossible (except error as second return value)")
	errVariadicTypeNotSupported = errors.New("variadic parameters are not supported")
	errIllegalVoidParameter     = errors.New("void is not a legal parameter type")
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

// This type is used to define the LD binding behavior
type Mode = dl.Mode

const (
	// Performs a lazy binding. Maps to RTLD_LAZY,
	// http://man7.org/linux/man-pages/man3/dlopen.3.html
	BindLazy = dl.Lazy

	// Performs a eager binding. Maps to RTLD_NOW,
	// http://man7.org/linux/man-pages/man3/dlopen.3.html
	BindNow = dl.Now

	// Makes symbols available locally. Maps to RTLD_LOCAL,
	// http://man7.org/linux/man-pages/man3/dlopen.3.html
	BindLocal = dl.Local

	// Makes symbols available globally. Maps to RTLD_GLOBAL,
	// http://man7.org/linux/man-pages/man3/dlopen.3.html
	BindGlobal = dl.Global
)

// This type represents the a loaded library, bound to a specific
// library file (.so or .dylib). All exported symbols of this
// library can be imported and mapped to Go functions.
type Library struct {
	lib         dl.Library
	name        string
	m           sync.Mutex
	cifCache    map[string]*C.ffi_cif
	symbolCache map[string]uintptr
}

// Loads a library file and create a Library instance bound to it.
// _library_ can be only the name, in which the library is searched
// in the library path LD_LIBRARY_PATH and working directory, or
// otherwise a relative or absolute path to the library file.
func NewLibrary(library string, mode Mode) (*Library, error) {
	path, err := dl.Find(library)
	if err != nil {
		return nil, err
	}
	library = path

	lib, err := dl.Open(library, mode)
	if err != nil {
		return nil, err
	}

	return &Library{
		lib:         lib,
		name:        library,
		cifCache:    make(map[string]*C.ffi_cif, 0),
		symbolCache: make(map[string]uintptr, 0),
	}, nil
}

// Closes the loaded Library. This is necessary to be called
// to clean internal state and the caches, which speeds up
// multiple requests for the same symbols.
func (l *Library) Close() error {
	for _, cif := range l.cifCache {
		if cif.arg_types != nil {
			C.free(unsafe.Pointer(cif.arg_types))
		}
	}
	return l.lib.Close()
}

// Retrieves the native function pointer to the requested
// symbol, or an error if the symbol is not found or any
// other problem occurred.
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

	l.symbolCache[name] = s
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

	// Meaningless since there is no Void in Go, still for documentation :)
	tt, err = cleanArgumentTypes(tt)
	if err != nil {
		return err
	}

	inTypes, inTypesPtr, nargs := wrapArgumentTypes(tt)

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

	cFnType := reflect.FuncOf(argumentTypes, []reflect.Type{retType}, false)

	out := []reflect.Type{retType}
	if returnsError {
		out = append(out, TypeError)
	}

	goFnType := reflect.FuncOf(argumentTypes, out, false)
	return l.NewImportComplex(symbol, goFnType, cFnType)
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
	cFnType, err = cleanArgumentTypes(cFnType)
	if err != nil {
		return nil, err
	}

	goFnType, err = cleanArgumentTypes(goFnType)
	if err != nil {
		return nil, err
	}

	inTypes, inTypesPtr, nargs := wrapArgumentTypes(cFnType)

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
	if fnType.IsVariadic() {
		return false, errVariadicTypeNotSupported
	}

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

func cleanArgumentTypes(fnType reflect.Type) (reflect.Type, error) {
	in := make([]reflect.Type, 0)
	out := make([]reflect.Type, 0)

	for i := 0; i < fnType.NumIn(); i++ {
		it := fnType.In(i)
		switch it {
		case TypeVoid:
			if i > 0 {
				return nil, errIllegalVoidParameter
			}
			continue
		}
		in = append(in, it)
	}

	for i := 0; i < fnType.NumOut(); i++ {
		it := fnType.Out(i)
		switch it {
		case TypeVoid:
			continue
		}
		out = append(out, it)
	}

	return reflect.FuncOf(in, out, fnType.IsVariadic()), nil
}

func wrapArgumentTypes(fnType reflect.Type) ([]ffiType, *ffiType, int) {
	nargs := fnType.NumIn()

	arguments := (*ffiType)(C.malloc(C.size_t(ptrSize * nargs)))

	header := reflect.SliceHeader{Data: uintptr(unsafe.Pointer(arguments)), Len: nargs, Cap: nargs}
	in := *(*[]ffiType)(unsafe.Pointer(&header))

	for i := 0; i < fnType.NumIn(); i++ {
		ot := fnType.In(i)
		in[i] = wrapType(ot)
	}
	return in, arguments, nargs
}

func wrapReturnType(fnType reflect.Type) ffiType {
	retType := typeVoid
	if fnType.NumOut() > 0 {
		retType = wrapType(fnType.Out(0))
	}
	return retType
}
