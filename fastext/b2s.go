package fastext

import "unsafe"

// B2s converts byte slice to a string without memory allocation.
// See https://groups.google.com/forum/#!msg/Golang-Nuts/ENgbUzYvCuU/90yGx7GUAgAJ .
func B2s(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}

func S2B(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(&s))
}
