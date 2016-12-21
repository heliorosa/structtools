package structtools_test

import (
	"bytes"

	"fmt"

	"github.com/heliorosa/structtools"
)

func ExampleEncoderDecoder() {
	type S struct {
		Id       int    `someTag:"+"`      // include
		Name     string `someTag:"aaaaaa"` // include
		SomeFlag bool   `someTag:"-"`      // "" or "-" ignores the field
	}
	data := S{1, "some Name", true}
	b := bytes.NewBuffer(make([]byte, 0, 256))
	if err := structtools.NewEncoderWithTags(b, "someTag", true).Encode(data); err != nil {
		panic(err)
	}
	unmarshaled := S{}
	if err := structtools.NewDecoder(b).Decode(&unmarshaled); err != nil {
		panic(err)
	}
	fmt.Printf("original data: %v\n", data)
	fmt.Printf("after unmarshaling: %v\n", unmarshaled)
	// Output: original data: {1 some Name true}
	// after unmarshaling: {1 some Name false}
}
