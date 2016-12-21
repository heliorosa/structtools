package structtools_test

import (
	"encoding/hex"
	"fmt"

	"github.com/heliorosa/structtools"
)

func ExampleMarshalUnmarshal() {
	type S struct {
		Id       int
		Name     string
		SomeFlag bool
	}
	data := S{1, "some Name", true}
	b, err := structtools.Marshal(data)
	if err != nil {
		panic(err)
	}
	unmarshaled := S{}
	sz, err := structtools.Unmarshal(b, &unmarshaled)
	if err != nil {
		panic(err)
	}
	fmt.Printf("parsed %d bytes of %d\n", sz, len(b))
	fmt.Printf("original data: %v\n", data)
	fmt.Printf("hex: %s\n", hex.EncodeToString(b))
	fmt.Printf("unmarshaled: %v\n", unmarshaled)
	// Output: parsed 22 bytes of 22
	// original data: {1 some Name true}
	// hex: 000000000000000100000009736f6d65204e616d6501
	// unmarshaled: {1 some Name true}
}
