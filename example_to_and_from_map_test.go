package structtools_test

import (
	"fmt"

	"github.com/heliorosa/structtools"
)

func ExampleToFromMap() {
	// our type
	type S struct {
		Id    int    `someTag:"id"`
		Name  string `someTag:"nombre"`
		Other bool
	}
	// some data
	data := S{1, "obvious name", true}
	// we only care about fields tagged with "someTag"
	mapData, err := structtools.ToMap("someTag", data, true)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%v:%v\n", mapData["id"], mapData["nombre"])
	// copy the data from the map to the struct
	otherData := &S{}
	if err = structtools.FromMap("someTag", mapData, otherData, true); err != nil {
		panic(err)
	}
	fmt.Printf("%v\n", data)
	fmt.Printf("%v\n", *otherData)
	// Output: 1:obvious name
	// {1 obvious name true}
	// {1 obvious name false}
}
