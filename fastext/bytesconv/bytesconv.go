package bytesconv

import (
	"unsafe"
)

func SToB(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

func BToS(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}
