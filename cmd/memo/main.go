package main

import (
	"encoding/json"
	"fmt"
	"reflect"
	"runtime"
	"time"

	"hash.reksie.com/internal/keys"
)

func MakeTimedFunction[T any](f T) T {
	//Get type of the function
	rf := reflect.TypeOf(f)

	//If not a function it just returns a error message
	if rf.Kind() != reflect.Func {
		panic("Expects function")
	}

	//Then MakeFunc call a function and return
	vf := reflect.ValueOf(f)
	wrapperF := reflect.MakeFunc(rf, func(fnArgs []reflect.Value) []reflect.Value {

		// Convert arguments to []interface{} for JSON marshaling
		args := make([]interface{}, len(fnArgs))
		for i, arg := range fnArgs {
			args[i] = arg.Interface()
		}

		// Marshal arguments to JSON
		jsonArgs, err := json.Marshal(args)
		if err != nil {
			fmt.Printf("Error marshaling arguments: %v\n", err)
		} else {
			fmt.Printf("Function arguments: %s\n", string(jsonArgs))
		}

		key, err := keys.HashKey(string(jsonArgs))
		if err != nil {
			fmt.Printf("Error hashing arguments: %v\n", err)
		}
		fmt.Printf("Function key: %s\n", key)

		//This will get start time of the function call
		start := time.Now()

		out := vf.Call(fnArgs)

		//It gets end time of the function call
		end := time.Now()

		fmt.Printf("Called it with %v\n", fnArgs)
		//it prints time different between two call
		fmt.Printf("Calling %s took %v \n", runtime.FuncForPC(vf.Pointer()).Name(), end.Sub(start))

		//then return output
		// It returns the output results as Values.
		return out
	})
	return wrapperF.Interface().(T)
}

func exampleString(x string) string {
	fmt.Println("Example function called with", x)
	return x
}

// Example usage
func main() {
	// Example function with one argument and one return value
	original1 := func(x int) int {
		fmt.Println("Original function called with", x)
		return x * 2
	}

	x := MakeTimedFunction(original1)

	x(5)

	timedString := MakeTimedFunction(exampleString)
	timedString("Hello")

	twoParams := func(x int, y string) (string, error) {

		if x == 5 {
			return "", fmt.Errorf("error: %d is not allowed", x)
		}

		fmt.Printf("Two params function called with %d and %s\n", x, y)
		return y, nil
	}

	timedTwoParams := MakeTimedFunction(twoParams)
	value, error := timedTwoParams(5, "Hello")
	if error != nil {
		fmt.Println("Oh no!:", error)
	}

	fmt.Println(value, error)

	timedTwoParams(10, "World")

}
