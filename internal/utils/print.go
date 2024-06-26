package utils

import (
	"fmt"
	"reflect"
)

func PrettyPrintStruct(s interface{}, level int, indent int) {
	fmt.Printf("%s{\n", getIndent((level-1)*indent))
	val := reflect.ValueOf(s)
	typ := reflect.TypeOf(s)
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)
		if field.Kind() == reflect.Struct {
			fmt.Printf("%s%s:\n", getIndent(level*indent), fieldType.Name)
			PrettyPrintStruct(field.Interface(), level+1, indent)
		} else if field.Kind() == reflect.Slice {
			fmt.Printf("%s%s:\n", getIndent(level*indent), fieldType.Name)
			fmt.Printf("%s[\n", getIndent(level*indent))
			for j := 0; j < field.Len(); j++ {
				PrettyPrintStruct(field.Index(j).Interface(), level+2, indent)
			}
			fmt.Printf("%s]\n", getIndent(level*indent))
		} else {
			fmt.Printf("%s%s: %v\n", getIndent(level*indent), fieldType.Name, field.Interface())
		}
	}
	fmt.Printf("%s}\n", getIndent((level-1)*indent))
}

func getIndent(indent int) string {
	indentStr := ""
	for i := 0; i < indent; i++ {
		indentStr += " "
	}
	return indentStr
}
