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

import (
	"testing"
)

const testLibrary = "target/libgoffitests"

func TestLoadLibraryFailed(t *testing.T) {
	_, err := NewLibrary("123libc", BindNow)
	if err == nil {
		t.Error("Library loading for lib 132libc shouldn't succeed")
	}
}

func TestLoadLibrary(t *testing.T) {
	l, err := NewLibrary("libc", BindNow)
	if err != nil {
		t.Errorf("Library failed to be initialized: %v", err)
	}
	l.Close()
}

func TestImportSymbol(t *testing.T) {
	l, err := NewLibrary("libc", BindNow)
	if err != nil {
		t.Errorf("Library failed to be initialized: %v", err)
	}
	var getpid func() int
	if err := l.Import("getpid", &getpid); err != nil {
		t.Errorf("Symbol getpid failed to be imported: %v", err)
	}
	l.Close()
}

func TestImportSymbolFailed(t *testing.T) {
	l, err := NewLibrary("libc", BindNow)
	if err != nil {
		t.Errorf("Library failed to be initialized: %v", err)
	}
	var abc func() int
	if err := l.Import("abc123", &abc); err == nil {
		t.Errorf("Symbol abc123 shouldn't be importable: %v", err)
	}
	l.Close()
}

func TestExecuteOneParamOneReturn(t *testing.T) {
	l, err := NewLibrary("target/libgoffitests.so", BindLazy)
	if err != nil {
		t.Errorf("Library failed to be initialized: %v", err)
	}
	var sqrt func(float64) float64
	if err := l.Import("_sqrt", &sqrt); err != nil {
		t.Errorf("Symbol sqrt failed to be imported: %v", err)
		return
	} else {
		ret := sqrt(9.)
		if ret != 3. {
			t.Fail()
		}
	}
	l.Close()
}

func TestExecuteNoParamOneReturn(t *testing.T) {
	l, err := NewLibrary("target/libgoffitests.so", BindNow)
	if err != nil {
		t.Errorf("Library failed to be initialized: %v", err)
	}
	var fn func() int
	if err := l.Import("_sint", &fn); err != nil {
		t.Errorf("Symbol _sint failed to be imported: %v", err)
	} else {
		fn()
	}
	l.Close()
}

func TestExecuteSint(t *testing.T) {
	var fn func() int
	libraryTestHelper(t, "_sint", testLibrary, &fn, func() {
		v := fn()
		if v != -1 {
			t.Fail()
		}
	})
}

func TestExecuteSint8(t *testing.T) {
	var fn func() int8
	libraryTestHelper(t, "_sint8", testLibrary, &fn, func() {
		v := fn()
		if v != -8 {
			t.Fail()
		}
	})
}

func TestExecuteSint16(t *testing.T) {
	var fn func() int16
	libraryTestHelper(t, "_sint16", testLibrary, &fn, func() {
		v := fn()
		if v != -16 {
			t.Fail()
		}
	})
}

func TestExecuteSint32(t *testing.T) {
	var fn func() int32
	libraryTestHelper(t, "_sint32", testLibrary, &fn, func() {
		v := fn()
		if v != -32 {
			t.Fail()
		}
	})
}

func TestExecuteSint64(t *testing.T) {
	var fn func() int64
	libraryTestHelper(t, "_sint64", testLibrary, &fn, func() {
		v := fn()
		if v != -64 {
			t.Fail()
		}
	})
}

func TestExecuteUint(t *testing.T) {
	var fn func() uint
	libraryTestHelper(t, "_uint", testLibrary, &fn, func() {
		v := fn()
		if v != 1 {
			t.Fail()
		}
	})
}

func TestExecuteUint8(t *testing.T) {
	var fn func() uint8
	libraryTestHelper(t, "_uint8", testLibrary, &fn, func() {
		v := fn()
		if v != 8 {
			t.Fail()
		}
	})
}

func TestExecuteUint16(t *testing.T) {
	var fn func() uint16
	libraryTestHelper(t, "_uint16", testLibrary, &fn, func() {
		v := fn()
		if v != 16 {
			t.Fail()
		}
	})
}

func TestExecuteUint32(t *testing.T) {
	var fn func() uint32
	libraryTestHelper(t, "_uint32", testLibrary, &fn, func() {
		v := fn()
		if v != 32 {
			t.Fail()
		}
	})
}

func TestExecuteUint64(t *testing.T) {
	var fn func() uint64
	libraryTestHelper(t, "_uint64", testLibrary, &fn, func() {
		v := fn()
		if v != 64 {
			t.Fail()
		}
	})
}

func TestExecuteFloat32(t *testing.T) {
	var fn func() float32
	libraryTestHelper(t, "_float", testLibrary, &fn, func() {
		v := fn()
		if v != 32.1 {
			t.Fail()
		}
	})
}

func TestExecuteFloat64(t *testing.T) {
	var fn func() float64
	libraryTestHelper(t, "_double", testLibrary, &fn, func() {
		v := fn()
		if v != -64.2 {
			t.Fail()
		}
	})
}

func TestExecuteIntInIntOut(t *testing.T) {
	var fn func(int) int
	libraryTestHelper(t, "__sint", testLibrary, &fn, func() {
		v := fn(2)
		if v != 1 {
			t.Fail()
		}
	})
}

func TestExecuteInt8InInt8Out(t *testing.T) {
	var fn func(int8) int8
	libraryTestHelper(t, "__sint8", testLibrary, &fn, func() {
		v := fn(15)
		if v != 7 {
			t.Fail()
		}
	})
}

func TestExecuteInt16InInt16Out(t *testing.T) {
	var fn func(int16) int16
	libraryTestHelper(t, "__sint16", testLibrary, &fn, func() {
		v := fn(31)
		if v != 15 {
			t.Fail()
		}
	})
}

func TestExecuteInt32InInt32Out(t *testing.T) {
	var fn func(int32) int32
	libraryTestHelper(t, "__sint32", testLibrary, &fn, func() {
		v := fn(63)
		if v != 31 {
			t.Fail()
		}
	})
}

func TestExecuteInt64InInt64Out(t *testing.T) {
	var fn func(int64) int64
	libraryTestHelper(t, "__sint64", testLibrary, &fn, func() {
		v := fn(127)
		if v != 63 {
			t.Fail()
		}
	})
}

func TestExecuteUintInUintOut(t *testing.T) {
	var fn func(uint) uint
	libraryTestHelper(t, "__uint", testLibrary, &fn, func() {
		v := fn(2)
		if v != 1 {
			t.Fail()
		}
	})
}

func TestExecuteUint8InUint8Out(t *testing.T) {
	var fn func(uint8) uint8
	libraryTestHelper(t, "__uint8", testLibrary, &fn, func() {
		v := fn(15)
		if v != 7 {
			t.Fail()
		}
	})
}

func TestExecuteUint16InUint16Out(t *testing.T) {
	var fn func(uint16) uint16
	libraryTestHelper(t, "__uint16", testLibrary, &fn, func() {
		v := fn(31)
		if v != 15 {
			t.Fail()
		}
	})
}

func TestExecuteUint32InUint32Out(t *testing.T) {
	var fn func(uint32) uint32
	libraryTestHelper(t, "__uint32", testLibrary, &fn, func() {
		v := fn(63)
		if v != 31 {
			t.Fail()
		}
	})
}

func TestExecuteUint64InUint64Out(t *testing.T) {
	var fn func(uint64) uint64
	libraryTestHelper(t, "__uint64", testLibrary, &fn, func() {
		v := fn(127)
		if v != 63 {
			t.Fail()
		}
	})
}

func TestExecuteCharPtrIntInCharPtrOut(t *testing.T) {
	var fn func(string, int) string
	libraryTestHelper(t, "_char", testLibrary, &fn, func() {
		msg := "hello world"
		v := fn(msg, len(msg) + 1)
		if v != "hello world" {
			t.Errorf("expected '%s', got '%s'", msg, v)
		}
	})
}

func libraryTestHelper(t *testing.T, symbol, library string, fn interface{}, test func()) {
	l, err := NewLibrary(library, BindNow)
	if err != nil {
		t.Errorf("Library %s failed to be initialized: %v", library, err)
		return
	}
	if err := l.Import(symbol, fn); err != nil {
		t.Errorf("Symbol %s failed to be imported: %v", symbol, err)
		return
	}
	test()
	l.Close()
}
