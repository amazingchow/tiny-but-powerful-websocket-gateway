package utils

import (
	"reflect"
	"testing"
)

func TestSafeJsonMarshalToString(t *testing.T) {
	// Test case 1
	val1 := struct {
		Name string
		Age  int
	}{
		Name: "John Doe",
		Age:  30,
	}
	expected1 := `{"Name":"John Doe","Age":30}`
	result1 := SafeJsonMarshalToString(val1)
	if result1 != expected1 {
		t.Errorf("Expected: %s, but got: %s", expected1, result1)
	}

	// Test case 2
	val2 := struct {
		Message string
	}{
		Message: "Hello, World!",
	}
	expected2 := `{"Message":"Hello, World!"}`
	result2 := SafeJsonMarshalToString(val2)
	if result2 != expected2 {
		t.Errorf("Expected: %s, but got: %s", expected2, result2)
	}

	// Add more test cases if needed
}

func TestSafeJsonUnmarshalFromString(t *testing.T) {
	// Test case 1
	content1 := `{"Name":"John Doe","Age":30}`
	type Struct1 struct {
		Name string
		Age  int
	}
	expected1 := Struct1{
		Name: "John Doe",
		Age:  30,
	}
	var result1 Struct1
	SafeJsonUnmarshalFromString(content1, &result1)
	if !reflect.DeepEqual(result1, expected1) {
		t.Errorf("Expected: %+v, but got: %+v", expected1, result1)
	}

	// Test case 2
	content2 := `{"Message":"Hello, World!"}`
	type Struct2 struct {
		Message string
	}
	expected2 := Struct2{
		Message: "Hello, World!",
	}
	var result2 Struct2
	SafeJsonUnmarshalFromString(content2, &result2)
	if !reflect.DeepEqual(result2, expected2) {
		t.Errorf("Expected: %+v, but got: %+v", expected2, result2)
	}

	// Add more test cases if needed
}
