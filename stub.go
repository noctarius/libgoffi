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

typedef void** argumentsPtr;

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

static void _ffi_call(ffi_cif *cif, void(*fn)(void), void *rvalue, void **values) {
	ffi_call(cif, fn, rvalue, values);
}
*/
import "C"
import (
	"errors"
	"fmt"
	"reflect"
	"unsafe"
)

var (
	ptrSize  = int(C._ptrSize)
	boolSize = int(C._boolSize)
	intSize  = int(C._intSize)
)

func makeStub(fnType reflect.Type, cif *C.ffi_cif, funcPtr functionPointer, outType ffiType,
	inTypes []ffiType, returnsError bool) func(values []reflect.Value) []reflect.Value {

	return func(values []reflect.Value) []reflect.Value {
		nargs := len(values)
		if nargs != len(inTypes) {
			msg := fmt.Sprintf("illegal argument length, expected %d, got %d", len(inTypes), nargs)
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

		var cargs C.argumentsPtr = nil
		if nargs > 0 {
			cargs = args
		}

		ot := unwrapType(outType)
		out := reflect.New(ot)
		_, err := C._ffi_call(cif, funcPtr, unsafe.Pointer(out.Elem().UnsafeAddr()), cargs)
		if err != nil {
			panic(err)
		}
		C.argsArrayFree(args)

		for i := 0; i < len(finalizers); i++ {
			finalizers[i]()
		}

		retValues := make([]reflect.Value, 0)
		if fnType.NumOut() > 0 {
			rt := fnType.Out(0)
			out = unwrapValue(out, rt)
			retValues = append(retValues, out)
		}

		if returnsError {
			retValues = append(retValues, valueNilError)
		}

		return retValues
	}
}
