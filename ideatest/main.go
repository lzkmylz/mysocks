package main

import (
	"fmt"
	"reflect"
)

func main() {
	a := make([]int, 10)
	fmt.Println(reflect.TypeOf(a))
}
