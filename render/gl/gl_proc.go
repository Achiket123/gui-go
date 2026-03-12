package gl

/*
// This file intentionally has a minimal preamble — CGo requires that any
// file using //export has NO non-declaration C code in its preamble.
// The actual GL function pointer declarations live in gl.go.
// Note: do NOT forward-declare getGLProcAddress here; CGo auto-generates
// the extern from the //export directive below.
*/
import "C"

import "unsafe"

// getProc is the Go-side function pointer loader, set by InitExtensions.
var getProc func(name string) unsafe.Pointer

// getGLProcAddress is the C-callable bridge exported so that the C code in
// gl.go can call back into Go to resolve OpenGL extension function pointers.
//
//export getGLProcAddress
func getGLProcAddress(name *C.char) unsafe.Pointer {
	if getProc != nil {
		return getProc(C.GoString(name))
	}
	return nil
}
