package libgoffi

import (
	"testing"
)

func TestLoadLibraryFailed(t *testing.T) {
	_, err := NewLibrary("123libc", BindNow)
	if err != nil {
		t.Errorf("Library failed to be initialized: %v", err)
	}
}

func TestLoadLibrary(t *testing.T) {
	_, err := NewLibrary("libc", BindNow)
	if err != nil {
		t.Errorf("Library failed to be initialized: %v", err)
	}
}
