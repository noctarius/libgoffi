package libgoffi

/*
#cgo LDFLAGS: -lffi
//#cgo pkg-config: libffi
#include <ffi.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>

const int _ptrSize = sizeof(uintptr_t);

const int _boolSize = sizeof(_Bool);

#if INT_MAX == 32767
const int _intSize = 2;
#elif INT_MAX == 2147483647
const int _intSize = 4;
#elif INT_MAX == 9223372036854775807
#error "64 bit base int is unsupported, please use int64 explicitly"
#endif

static void **argsArrayNew(int nargs) {
	void** ptr = (void **)(malloc(nargs * _ptrSize));
	memset(ptr, 0, nargs * _ptrSize);
	return ptr;
}

static void argsArraySet(void **args, int index, uintptr_t ptr) {
	args[index] = (void *) ptr;
}

static void argsArrayFree(void **args) {
	free(args);
}

typedef void (*fnptr_t)(void);
typedef void* _pointer;
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

type pointer C._pointer

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
	argumentTypes ...reflect.Type) (interface{}, error) {

	out := []reflect.Type{retType}
	if returnsError {
		out = append(out, TypeError)
	}

	funcType := reflect.FuncOf(argumentTypes, out, false)
	return l.GoFunction(symbol, funcType, funcType)
}

func (l *Library) GoFunction(symbol string, goFnType reflect.Type, cFnType reflect.Type) (interface{}, error) {
	if goFnType.Kind() != reflect.Func {
		return nil, errNoGoFuncDef
	}
	if cFnType.Kind() != reflect.Func {
		return nil, errNoCFuncDef
	}

	returnsError := false
	if goFnType.NumOut() > 1 {
		if goFnType.NumOut() > 2 {
			return nil, errGoFuncMultiReturn
		}
		if !goFnType.Out(1).Implements(TypeError) {
			return nil, errGoFuncMultiReturn
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
		return nil, err
	}

	sym, err := l.lib.Symbol(symbol)
	if err != nil {
		return nil, err
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

		//args := makeArgsArray(nargs) // make([]unsafe.Pointer, nargs) //makeArgsArray(nargs)
		args := C.argsArrayNew(C.int(nargs))
		finalizers := make([]finalizer, 0)
		for i := 0; i < nargs; i++ {
			//arg, fin := wrapValue(values[i])

			var fin finalizer = nil

			value := values[i]
			t := value.Type()
			v := value.Interface()
			switch t.Kind() {
			case reflect.String:
				cs := C.CString(v.(string))
				C.argsArraySet(args, C.int(i), C.ulong(uintptr(unsafe.Pointer(&cs))))
				//args[i] = unsafe.Pointer(&cs)
				fin = func() {
					C.free(unsafe.Pointer(&cs))
				}

			case reflect.UnsafePointer:
				//args[i] = v.(unsafe.Pointer)
			case reflect.Uintptr:
				//args[i] = unsafe.Pointer(v.(uintptr))
			case reflect.Uint:
				/*val := value.Uint()
				if intSize == 2 {
					v := C.uint16_t(val)
					args[i] = unsafe.Pointer(&v)
				} else {
					v := C.uint32_t(val)
					args[i] = unsafe.Pointer(&v)
				}*/
				//args[i] = unsafe.Pointer(value.Elem().Pointer())
			case reflect.Uint8:
				//val := C.uint8_t(value.Uint())
				//args[i] = unsafe.Pointer(&val)
			case reflect.Uint16:
				//val := C.uint16_t(value.Uint())
				//args[i] = unsafe.Pointer(&val)
			case reflect.Uint32:
				//val := C.uint32_t(value.Uint())
				//args[i] = unsafe.Pointer(&val)
			case reflect.Uint64:
				//val := C.uint64_t(value.Uint())
				//args[i] = unsafe.Pointer(&val)
			case reflect.Int:
				/*val := value.Int()
				if intSize == 2 {
					v := C.int16_t(val)
					args[i] = unsafe.Pointer(&v)
				} else {
					v := C.int32_t(val)
					args[i] = unsafe.Pointer(&v)
				}*/
				ptr := reflect.New(value.Type())
				ptr.Elem().Set(value)
				//args[i] = ptr.Elem().UnsafeAddr()
				C.argsArraySet(args, C.int(i), C.ulong(ptr.Elem().UnsafeAddr()))
			case reflect.Int8:
				//val := C.int8_t(value.Int())
				//args[i] = unsafe.Pointer(&val)
			case reflect.Int16:
				//val := C.int16_t(value.Int())
				//args[i] = unsafe.Pointer(&val)
			case reflect.Int32:
				//val := C.int32_t(value.Int())
				//args[i] = unsafe.Pointer(&val)
			case reflect.Int64:
				//val := C.int64_t(value.Int())
				//args[i] = unsafe.Pointer(&val)
			case reflect.Float32:
				//val := C.float(value.Float())
				//args[i] = unsafe.Pointer(&val)
			case reflect.Float64:
				//val := C.double(value.Float())
				//args[i] = unsafe.Pointer(&val)
			case reflect.Ptr:
				//args[i] = unsafe.Pointer(value.Elem().UnsafeAddr())
			case reflect.Bool:
				/*b := 0
				if v.(bool) {
					b = 1
				}

				if boolSize == 1 {
					val := C.int8_t(b)
					args[i] = unsafe.Pointer(&val)
				} else {
					val := C.int16_t(b)
					args[i] = unsafe.Pointer(&val)
				}*/
			}

			if fin != nil {
				finalizers = append(finalizers, fin)
			}
		}

		ot := unwrapType(retType)
		out := reflect.New(ot)
		if nargs > 0 {
			_, err := C.ffi_call(&cif, fnPtr, unsafe.Pointer(out.Elem().UnsafeAddr()), args)
			if err != nil {
				panic(err)
			}
			//argsPtr = (*unsafe.Pointer)(&args[0]) //argsArrayPointer(args) //&args[0]
		} else {
			_, err := C.ffi_call(&cif, fnPtr, unsafe.Pointer(out.Elem().UnsafeAddr()), nil)
			if err != nil {
				panic(err)
			}
		}
		//freeArgsArray(args)
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

	return reflect.MakeFunc(goFnType, stub).Interface(), nil
}

func newCif(fnType reflect.Type) (C.ffi_cif, error) {
	var cif C.ffi_cif

	nargs := fnType.NumIn()
	args := make([]ffiType, nargs)
	for i := 0; i < nargs; i++ {
		args[i] = wrapType(fnType.In(i))
	}

	ret := typeVoid
	if fnType.NumOut() > 0 {
		ret = wrapType(fnType.Out(0))
	}

	var argsPtr *ffiType = nil
	if len(args) > 0 {
		argsPtr = &args[0]
	}

	retval := status(C.ffi_prep_cif(&cif, C.FFI_DEFAULT_ABI, C.uint(nargs), ret, argsPtr))
	if retval != ffiOk {
		return cif, retval
	}

	return cif, nil
}

func makeArgsArray(nargs int) []pointer {
	carr := (**C.void)(C.malloc(C.ulong(nargs * ptrSize)))
	header := reflect.SliceHeader{Data: uintptr(unsafe.Pointer(carr)), Len: nargs, Cap: nargs}
	return *(*[]pointer)(unsafe.Pointer(&header))
}

func argsArrayPointer(slice []pointer) *pointer {
	header := (*reflect.SliceHeader)(unsafe.Pointer(&slice))
	return (*pointer)(unsafe.Pointer(header.Data))
}

func freeArgsArray(slice []pointer) {
	header := (*reflect.SliceHeader)(unsafe.Pointer(&slice))
	C.free(unsafe.Pointer(header.Data))
}
