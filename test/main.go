package main

import "fmt"

func Test[Value int | string](a Value) Value {
	fmt.Println(a)
	return a
}

func main() {
	Test(1)
	Test("123")
	// g
}
