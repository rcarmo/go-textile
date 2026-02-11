package gotextile

import "fmt"

func ExampleTextileToHtml() {
	input := "h1. Hello"
	output, _ := TextileToHtml(input)
	fmt.Print(output)
	// Output:
	// <h1>Hello</h1>
}
