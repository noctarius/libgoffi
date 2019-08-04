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
#include <stdint.h>
*/
import "C"
import (
	"errors"
	"fmt"
	"reflect"
	"unsafe"
)

type ffiType = *C.ffi_type

var (
	typeVoid    ffiType = &C.ffi_type_void
	typeUint8   ffiType = &C.ffi_type_uint8
	typeUint16  ffiType = &C.ffi_type_uint16
	typeUint32  ffiType = &C.ffi_type_uint32
	typeUint64  ffiType = &C.ffi_type_uint64
	typeInt8    ffiType = &C.ffi_type_sint8
	typeInt16   ffiType = &C.ffi_type_sint16
	typeInt32   ffiType = &C.ffi_type_sint32
	typeInt64   ffiType = &C.ffi_type_sint64
	typeFloat   ffiType = &C.ffi_type_float
	typeDouble  ffiType = &C.ffi_type_double
	typePointer ffiType = &C.ffi_type_pointer
)

var (
	TypeError   = reflect.TypeOf((*error)(nil)).Elem()
	TypeUint    = reflect.TypeOf(uint(0))
	TypeUint8   = reflect.TypeOf(uint8(0))
	TypeUint16  = reflect.TypeOf(uint16(0))
	TypeUint32  = reflect.TypeOf(uint32(0))
	TypeUint64  = reflect.TypeOf(uint64(0))
	TypeInt     = reflect.TypeOf(int(0))
	TypeInt8    = reflect.TypeOf(int8(0))
	TypeInt16   = reflect.TypeOf(int16(0))
	TypeInt32   = reflect.TypeOf(int32(0))
	TypeInt64   = reflect.TypeOf(int64(0))
	TypeFloat32 = reflect.TypeOf(float32(0))
	TypeFloat64 = reflect.TypeOf(float64(0))
	TypeUintptr = reflect.TypeOf(uintptr(0))
	TypeVoid    = reflect.TypeOf(&struct{}{})
)

var valueNil = reflect.ValueOf(nil)
var valueNilError = reflect.Zero(TypeError)

func wrapType(t reflect.Type) ffiType {
	// Unhandled for now
	// - map
	// - slices (maybe only pointer slices?)
	// - func
	// - interface
	// - array
	// - struct
	// - complex64
	// - complex128
	// - chan

	switch t.Kind() {
	case reflect.String:
		fallthrough
	case reflect.Ptr:
		fallthrough
	case reflect.UnsafePointer:
		fallthrough
	case reflect.Uintptr:
		return typePointer

	case reflect.Uint:
		if intSize == 2 {
			return typeUint16
		}
		return typeUint32
	case reflect.Uint8:
		return typeUint8
	case reflect.Uint16:
		return typeUint16
	case reflect.Uint32:
		return typeUint32
	case reflect.Uint64:
		return typeUint64

	case reflect.Int:
		if intSize == 2 {
			return typeInt16
		}
		return typeInt32
	case reflect.Int8:
		return typeInt8
	case reflect.Int16:
		return typeInt16
	case reflect.Int32:
		return typeInt32
	case reflect.Int64:
		return typeInt64

	case reflect.Float32:
		return typeFloat
	case reflect.Float64:
		return typeDouble

	case reflect.Bool:
		if boolSize == 1 {
			return typeInt8
		}
		return typeInt16
	}
	panic(errors.New(fmt.Sprintf("unhandled data type: %s", t.Kind().String())))
}

func unwrapType(t ffiType) reflect.Type {
	switch t {
	case typeVoid:
		return TypeVoid

	case typeUint8:
		return TypeUint8
	case typeUint16:
		return TypeUint16
	case typeUint32:
		return TypeUint32
	case typeUint64:
		return TypeUint64

	case typeInt8:
		return TypeInt8
	case typeInt16:
		return TypeInt16
	case typeInt32:
		return TypeInt32
	case typeInt64:
		return TypeInt64

	case typeFloat:
		return TypeFloat32
	case typeDouble:
		return TypeFloat64

	case typePointer:
		return TypeUintptr
	}
	panic(errors.New(fmt.Sprintf("unhandled data type: %d", t)))
}

type finalizer = func()

func wrapValue(value reflect.Value) (unsafe.Pointer, finalizer) {
	t := value.Type()
	v := value.Interface()
	switch t.Kind() {
	case reflect.String:
		cs := C.CString(v.(string))
		fin := func() {
			C.free(unsafe.Pointer(cs))
		}
		return unsafe.Pointer(cs), fin

	case reflect.UnsafePointer:
		return v.(unsafe.Pointer), nil
	case reflect.Uintptr:
		return unsafe.Pointer(v.(uintptr)), nil

	case reflect.Uint:
		val := value.Uint()
		if intSize == 2 {
			v := C.uint16_t(val)
			return unsafe.Pointer(&v), nil
		}
		v := C.uint32_t(val)
		return unsafe.Pointer(&v), nil

	case reflect.Uint8:
		val := C.uint8_t(value.Uint())
		return unsafe.Pointer(&val), nil

	case reflect.Uint16:
		val := C.uint16_t(value.Uint())
		return unsafe.Pointer(&val), nil

	case reflect.Uint32:
		val := C.uint32_t(value.Uint())
		return unsafe.Pointer(&val), nil

	case reflect.Uint64:
		val := C.uint64_t(value.Uint())
		return unsafe.Pointer(&val), nil

	case reflect.Int:
		val := value.Int()
		if intSize == 2 {
			v := C.int16_t(val)
			return unsafe.Pointer(&v), nil
		}
		v := C.int32_t(val)
		return unsafe.Pointer(&v), nil

	case reflect.Int8:
		val := C.int8_t(value.Int())
		return unsafe.Pointer(&val), nil

	case reflect.Int16:
		val := C.int16_t(value.Int())
		return unsafe.Pointer(&val), nil

	case reflect.Int32:
		val := C.int32_t(value.Int())
		return unsafe.Pointer(&val), nil

	case reflect.Int64:
		val := C.int64_t(value.Int())
		return unsafe.Pointer(&val), nil

	case reflect.Float32:
		val := C.float(value.Float())
		return unsafe.Pointer(&val), nil

	case reflect.Float64:
		val := C.double(value.Float())
		return unsafe.Pointer(&val), nil

	case reflect.Ptr:
		return unsafe.Pointer(value.Elem().UnsafeAddr()), nil

	case reflect.Bool:
		b := 0
		if v.(bool) {
			b = 1
		}

		if boolSize == 1 {
			val := C.int8_t(b)
			return unsafe.Pointer(&val), nil
		} else {
			val := C.int16_t(b)
			return unsafe.Pointer(&val), nil
		}
	}
	panic(errors.New(fmt.Sprintf("unhandled data type: %s", t.Kind().String())))
}

func unwrapValue(value reflect.Value, t reflect.Type) reflect.Value {
	vt := value.Type()
	if vt.Kind() == reflect.Ptr {
		ivt := reflect.Indirect(value)
		if t.Kind() == reflect.Ptr {
			it := t.Elem()
			if ivt.Kind() == it.Kind() {
				return value
			}
		}

		value = ivt
		vt = value.Type()
	}

	if t.Kind() == reflect.Ptr {
		it := t.Elem()
		reflect.ValueOf(value.Convert(it).Interface())
		pv := reflect.New(it)
		pv.Elem().Set(value)
		value = pv
	} else {
		value = reflect.ValueOf(value.Convert(t).Interface())
	}

	return value
}
