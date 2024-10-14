package internal

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// Memoize takes a function with any number of arguments and returns a memoized version of that function
func Memoize(f interface{}) interface{} {
	ft := reflect.TypeOf(f)
	if ft.Kind() != reflect.Func {
		panic("Memoize: argument must be a function")
	}

	cache := make(map[string][]reflect.Value)

	return reflect.MakeFunc(ft, func(args []reflect.Value) []reflect.Value {
		key, err := hashArgs(args)
		if err != nil {
			panic(fmt.Sprintf("Memoize: failed to hash arguments: %v", err))
		}

		if cached, found := cache[key]; found {
			return cached
		}

		results := reflect.ValueOf(f).Call(args)
		cache[key] = results
		return results
	}).Interface()
}

// hashArgs creates a string hash of the given arguments
func hashArgs(args []reflect.Value) (string, error) {
	var values []interface{}
	for _, arg := range args {
		values = append(values, arg.Interface())
	}
	bytes, err := json.Marshal(values)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// ToComparable converts a value to a comparable type for use as a map key
func ToComparable(v interface{}) interface{} {
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Array, reflect.Slice, reflect.Map, reflect.Struct:
		return fmt.Sprintf("%#v", v)
	default:
		return v
	}
}
