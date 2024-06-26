package utils

import "unsafe"

// String2ByteSlice converts a string to a byte slice in a safe and efficient way.
func String2ByteSlice(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(&s))
}

// ByteSlice2String converts a byte slice to a string in a safe and efficient way.
func ByteSlice2String(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}
