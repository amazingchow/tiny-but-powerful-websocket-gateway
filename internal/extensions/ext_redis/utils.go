package extredis

import (
	"github.com/vmihailenco/msgpack/v5"
)

func Marshal(value interface{}) ([]byte, error) {
	switch value := value.(type) {
	case nil:
		return nil, nil
	case []byte:
		return value, nil
	case string:
		return []byte(value), nil
	}
	return msgpack.Marshal(value)
}

func Unmarshal(b []byte, value interface{}) error {
	n := len(b)
	if n == 0 {
		return nil
	}

	switch value := value.(type) {
	case nil:
		return nil
	case *[]byte:
		clone := make([]byte, n)
		copy(clone, b)
		*value = clone
		return nil
	case *string:
		*value = string(b)
		return nil
	}

	return msgpack.Unmarshal(b, value)
}
