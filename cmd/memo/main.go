package main

import (
	"fmt"
	"reflect"
	"runtime"
	"time"
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
	wrapperF := reflect.MakeFunc(rf, func(in []reflect.Value) []reflect.Value {

		//This will get start time of the function call
		start := time.Now()

		out := vf.Call(in)

		//It gets end time of the function call
		end := time.Now()

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

}
