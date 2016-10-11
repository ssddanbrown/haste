package main

import (
	"fmt"
	"github.com/ssddanbrown/het/engine"
)

var testInput string = `

<div>
	<t:hello>
	this is some content
		<t:test>
			more content
		</t:test>
	</t:hello>
a
</div>

`

func main() {

	o, err := engine.ParseString(testInput)
	if err != nil {
		fmt.Println("error: " + err.Error())
	}
	fmt.Println(o)
}
