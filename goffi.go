package libgoffi

/*
//#cgo LDFLAGS: -lffi
#cgo pkg-config: libffi
#include <ffi.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>
#include <stdio.h>

const int _ptrSize = sizeof(uintptr_t);

const int _boolSize = sizeof(_Bool);

#if INT_MAX == 32767
const int _intSize = 2;
#elif INT_MAX == 2147483647
const int _intSize = 4;
#elif INT_MAX == 9223372036854775807
#error "64 bit base int is unsupported, please use int64 explicitly"
#endif

typedef void (*fnptr_t)(void);
typedef void* _pointer;
typedef void** arguments;

static void **argsArrayNew(int nargs) {
	void** ptr = (void **)(malloc(nargs * _ptrSize));
	memset(ptr, 0, nargs * _ptrSize);
	return ptr;
}

static void argsArraySet(void **args, int index, void *ptr) {
	args[index] = ptr;
}

static void argsArrayFree(void **args) {
	free(args);
}

static void _ffi_call(ffi_cif *cif, fnptr_t fn, void *rvalue, void **values) {
	printf("test");
	ffi_call(cif, fn, rvalue, values);
}

static void *makeInt(int val) {
	void *ptr = (void *) malloc(sizeof(int));
	*((int*) ptr) = val;
	return ptr;
}
*/
import "C"

import (
	"errors"
	"fmt"
	"github.com/achille-roussel/go-dl"
	"reflect"
	"strings"
	"unsafe"
)

var (
	errNoGoFuncDef       = errors.New("parameter is not a Go function definition")
	errNoCFuncDef        = errors.New("parameter is not a C function definition")
	errGoFuncMultiReturn = errors.New("multiple return values for Go impossible (except error as second return value)")
)

var (
	ptrSize  = int(C._ptrSize)
	boolSize = int(C._boolSize)
	intSize  = int(C._intSize)
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

type FFICleaner func()

type Library struct {
	lib dl.Library
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

	return &Library{lib: lib}, nil
}

func (l *Library) Close() error {
	return l.lib.Close()
}

func (l *Library) Function(symbol string, retType reflect.Type, returnsError bool,
	argumentTypes ...reflect.Type) (interface{}, FFICleaner, error) {

	out := []reflect.Type{retType}
	if returnsError {
		out = append(out, TypeError)
	}

	funcType := reflect.FuncOf(argumentTypes, out, false)
	return l.GoFunction(symbol, funcType, funcType)
}

func (l *Library) GoFunction(symbol string, goFnType reflect.Type, cFnType reflect.Type) (interface{}, FFICleaner, error) {
	if goFnType.Kind() != reflect.Func {
		return nil, nil, errNoGoFuncDef
	}
	if cFnType.Kind() != reflect.Func {
		return nil, nil, errNoCFuncDef
	}

	returnsError := false
	if goFnType.NumOut() > 1 {
		if goFnType.NumOut() > 2 {
			return nil, nil, errGoFuncMultiReturn
		}
		if !goFnType.Out(1).Implements(TypeError) {
			return nil, nil, errGoFuncMultiReturn
		}
		returnsError = true
	}

	in := make([]ffiType, 0)
	for i := 0; i < cFnType.NumIn(); i++ {
		ot := cFnType.In(i)
		t := wrapType(ot)
		switch ot {
		case TypeVoid:
			continue
		}
		in = append(in, t)
	}

	cif, err := newCif(cFnType)
	if err != nil {
		return nil, nil, err
	}

	var cleaner FFICleaner = func() {
		C.free(unsafe.Pointer(cif.arg_types))
	}

	sym, err := l.lib.Symbol(symbol)
	if err != nil {
		cleaner()
		return nil, nil, err
	}

	fnPtr := (C.fnptr_t)(unsafe.Pointer(sym))
	retType := typeVoid
	if cFnType.NumOut() > 0 {
		retType = wrapType(cFnType.Out(0))
	}

	stub := func(values []reflect.Value) []reflect.Value {
		nargs := len(values)
		if nargs != len(in) {
			msg := fmt.Sprintf("illegal argument length, expected %d, got %d", len(in), nargs)
			err := errors.New(msg)

			if returnsError {
				return []reflect.Value{valueNil, reflect.ValueOf(err)}
			}
			panic(err)
		}

		args := C.argsArrayNew(C.int(nargs))
		finalizers := make([]finalizer, 0)
		for i := 0; i < nargs; i++ {
			arg, fin := wrapValue(values[i])

			if fin != nil {
				finalizers = append(finalizers, fin)
			}
			C.argsArraySet(args, C.int(i), arg)
		}

		var cargs C.arguments = nil
		if nargs > 0 {
			cargs = args
		}

		ot := unwrapType(retType)
		out := reflect.New(ot)
		_, err := C._ffi_call(&cif, fnPtr, unsafe.Pointer(out.Elem().UnsafeAddr()), cargs)
		if err != nil {
			panic(err)
		}
		C.argsArrayFree(args)

		for i := 0; i < len(finalizers); i++ {
			finalizers[i]()
		}

		retValues := make([]reflect.Value, 0)
		if goFnType.NumOut() > 0 {
			rt := goFnType.Out(0)
			out = unwrapValue(out, rt)
			retValues = append(retValues, out)
		}

		if returnsError {
			retValues = append(retValues, valueNilError)
		}

		return retValues
	}

	return reflect.MakeFunc(goFnType, stub).Interface(), cleaner, nil
}

func newCif(fnType reflect.Type) (C.ffi_cif, error) {
	var cif C.ffi_cif

	nargs := fnType.NumIn()
	args := (*ffiType)(C.malloc(C.size_t(ptrSize * nargs)))

	header := reflect.SliceHeader{Data: uintptr(unsafe.Pointer(args)), Len: nargs, Cap: nargs}
	slice := *(*[]ffiType)(unsafe.Pointer(&header))

	for i := 0; i < nargs; i++ {
		slice[i] = wrapType(fnType.In(i))
	}

	ret := typeVoid
	if fnType.NumOut() > 0 {
		ret = wrapType(fnType.Out(0))
	}

	var argsPtr *ffiType = nil
	if len(slice) > 0 {
		argsPtr = args
	}

	retval := status(C.ffi_prep_cif(&cif, C.FFI_DEFAULT_ABI, C.uint(nargs), ret, argsPtr))
	if retval != ffiOk {
		return cif, retval
	}

	return cif, nil
}
