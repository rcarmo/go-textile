package gotextile

import "fmt"

func ExampleTextileToHtmlWithOptions() {
	options := DefaultOptions()
	options.HTML5 = true
	output, _ := TextileToHtmlWithOptions("p. Hi", options)
	fmt.Print(output)
	// Output:
	// <p>Hi</p>
}
