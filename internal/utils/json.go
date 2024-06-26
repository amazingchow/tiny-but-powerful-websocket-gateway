package utils

import (
	"github.com/bytedance/sonic"
)

// SafeJsonMarshal marshals the given value into a JSON byte slice.
func SafeJsonMarshal(val interface{}) []byte {
	buf, _ := sonic.Marshal(val)
	return buf
}

// SafeJsonMarshalToString marshals the given value into a JSON string.
func SafeJsonMarshalToString(val interface{}) string {
	buf, _ := sonic.Marshal(val)
	return ByteSlice2String(buf)
}

// SafeJsonUnmarshal unmarshals the given JSON byte slice into the given value.
func SafeJsonUnmarshal(buf []byte, val interface{}) {
	_ = sonic.Unmarshal(buf, val)
}

// SafeJsonUnmarshalFromString unmarshals the given JSON string into the given value.
func SafeJsonUnmarshalFromString(content string, val interface{}) {
	_ = sonic.Unmarshal(String2ByteSlice(content), val)
}
