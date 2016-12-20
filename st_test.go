package structtools

import (
	"encoding/binary"
	"encoding/hex"
	"io"
	"strings"
	"testing"
	"unsafe"
)

const (
	testTag = "test"
	valOfA  = 123
)

var (
	someStr  = "yada"
	otherStr = "hello world"
)

type MyStruct struct {
	A int     `test:"fieldA"`
	B string  `test:"b"`
	C *string `test:"cc"`
	D bool    `test:"-"`
	E string
}

func TestToAndFromMap(t *testing.T) {
	m := map[string]interface{}{
		"fieldA": valOfA,
		"b":      someStr,
		"cc":     &someStr,
		"E":      otherStr,
	}
	_ = m
	out := &MyStruct{}
	if err := FromMap(testTag, m, out, false); err != nil {
		t.Error(err)
		return
	}
	if out.A != valOfA {
		t.Error("got a different value for field A")
		return
	}
	if out.B != someStr {
		t.Error("got a different value for field B")
		return
	}
	if out.C != &someStr {
		t.Error("got a different value for field C")
		return
	}
	if out.E != otherStr {
		t.Error("got a different value for field C")
		return
	}
	m, err := ToMap(testTag, out, false)
	if err != nil {
		t.Error(err)
		return
	}
	if val, ok := m["fieldA"]; !ok {
		t.Error(`missing "fieldA" in map`)
		return
	} else if v, ok := val.(int); !ok {
		t.Error("expecting an int")
		return
	} else if v != valOfA {
		t.Error("got a different value than expected")
		return
	}
	if val, ok := m["b"]; !ok {
		t.Error(`missing "b" in map`)
		return
	} else if v, ok := val.(string); !ok {
		t.Error("expecting a string")
		return
	} else if v != someStr {
		t.Error("got a different value than expected")
		return
	}
	if val, ok := m["cc"]; !ok {
		t.Error(`missing "cc" in map`)
		return
	} else if v, ok := val.(*string); !ok {
		t.Error("expecting a pointer to a string")
		return
	} else if v != &someStr {
		t.Error("got a different value than expected")
		return
	}
}

type myInt int

func (i myInt) MarshalBinary(w io.Writer) (int, error) {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(i))
	if err := writeAll(w, b); err != nil {
		return 0, err
	}
	return len(b), nil
}

func (i *myInt) UnmarshalBinary(r io.Reader) (int, error) {
	b, err := readN(r, 8)
	if err != nil {
		return 0, err
	}
	*i = myInt(binary.BigEndian.Uint64(b))
	return 8, nil
}

type _test struct {
	expHex string
	v      interface{}
}

var (
	s = MyStruct{
		A: 1,
		B: someStr,
		C: &someStr,
		D: true,
	}
	hexS = strings.Join([]string{
		"0000000000000001",
		"0000000479616461",
		"0000000479616461",
		"01",
		"00000000",
	}, "")
	mustPassTests = []_test{
		{"", io.Reader(nil)},
		{"", (*MyStruct)(nil)},
		{"01", byte(1)},
		{"01", int8(1)},
		{"0002", int16(2)},
		{"00000003", int32(3)},
		{"0000000000000004", int64(4)},
		{"0000000000000005", int(5)},
		{"06", uint8(6)},
		{"0007", uint16(7)},
		{"00000008", uint32(8)},
		{"0000000000000009", uint64(9)},
		{"000000000000000a", uint(10)},
		{"000000000000000b", myInt(11)},
		{"01", true},
		{"4048" + "f5c3", float32(3.14)},
		{"40091eb8" + "51eb851f", float64(3.14)},
		{"42f60000" + "00000000", complex64(123)},
		{"405ec00000000000" + "0000000000000000", complex128(123)},
		{"0000000a" + "74657374537472696e67", "testString"},
		{hexS, s},
		{hexS, &s},
		{"00000001000200030004", [5]uint16{0, 1, 2, 3, 4}},
		{"0000000500000001000200030004", []uint16{0, 1, 2, 3, 4}},
		{"00000001" + "00000001610001", map[string]uint16{"a": 1}},
	}
	mustFailTests = []_test{
		{"", make(chan int, 1)},
		{"", func() {}},
		{"", unsafe.Pointer(uintptr(0))},
		{"", uintptr(0)},
	}
)

func TestEncode(t *testing.T) {
	for i, test := range mustPassTests {
		t.Log("encoding", test.expHex, test.v)
		b, err := Marshal(test.v)
		if err != nil {
			t.Error("can't marshal:", err)
			return
		}
		if xs := hex.EncodeToString(b); xs != test.expHex {
			t.Errorf("got different values (test %d): expected: %s got: %s\n", i, test.expHex, xs)
			return
		}
	}

	for _, test := range mustFailTests {
		_, err := Marshal(test.v)
		if err == nil {
			t.Error("expecting an error")
			return
		}
	}
}

func TestDecode(t *testing.T) {
	idx := 3
	b, _ := hex.DecodeString(mustPassTests[idx].expHex)
	var v1 int8
	if _, err := Unmarshal(b, &v1); err != nil {
		t.Error(err)
		return
	}
	if mustPassTests[idx].v.(int8) != v1 {
		t.Error("got different values")
		return
	}

	idx++
	b, _ = hex.DecodeString(mustPassTests[idx].expHex)
	var v2 int16
	if _, err := Unmarshal(b, &v2); err != nil {
		t.Error(err)
		return
	}
	if mustPassTests[idx].v.(int16) != v2 {
		t.Error("got different values")
		return
	}

	idx++
	b, _ = hex.DecodeString(mustPassTests[idx].expHex)
	var v3 int32
	if _, err := Unmarshal(b, &v3); err != nil {
		t.Error(err)
		return
	}
	if mustPassTests[idx].v.(int32) != v3 {
		t.Error("got different values")
		return
	}

	idx++
	b, _ = hex.DecodeString(mustPassTests[idx].expHex)
	var v4 int64
	if _, err := Unmarshal(b, &v4); err != nil {
		t.Error(err)
		return
	}
	if mustPassTests[idx].v.(int64) != v4 {
		t.Error("got different values")
		return
	}

	idx++
	b, _ = hex.DecodeString(mustPassTests[idx].expHex)
	var v5 int
	if _, err := Unmarshal(b, &v5); err != nil {
		t.Error(err)
		return
	}
	if mustPassTests[idx].v.(int) != v5 {
		t.Error("got different values")
		return
	}

	idx++
	b, _ = hex.DecodeString(mustPassTests[idx].expHex)
	var v6 uint8
	if _, err := Unmarshal(b, &v6); err != nil {
		t.Error(err)
		return
	}
	if mustPassTests[idx].v.(uint8) != v6 {
		t.Error("got different values")
		return
	}

	idx++
	b, _ = hex.DecodeString(mustPassTests[idx].expHex)
	var v7 uint16
	if _, err := Unmarshal(b, &v7); err != nil {
		t.Error(err)
		return
	}
	if mustPassTests[idx].v.(uint16) != v7 {
		t.Error("got different values")
		return
	}

	idx++
	b, _ = hex.DecodeString(mustPassTests[idx].expHex)
	var v8 uint32
	if _, err := Unmarshal(b, &v8); err != nil {
		t.Error(err)
		return
	}
	if mustPassTests[idx].v.(uint32) != v8 {
		t.Error("got different values")
		return
	}

	idx++
	b, _ = hex.DecodeString(mustPassTests[idx].expHex)
	var v9 uint64
	if _, err := Unmarshal(b, &v9); err != nil {
		t.Error(err)
		return
	}
	if mustPassTests[idx].v.(uint64) != v9 {
		t.Error("got different values")
		return
	}

	idx++
	b, _ = hex.DecodeString(mustPassTests[idx].expHex)
	var v10 uint
	if _, err := Unmarshal(b, &v10); err != nil {
		t.Error(err)
		return
	}
	if mustPassTests[idx].v.(uint) != v10 {
		t.Error("got different values")
		return
	}

	idx++
	b, _ = hex.DecodeString(mustPassTests[idx].expHex)
	var v11 myInt
	if _, err := Unmarshal(b, &v11); err != nil {
		t.Error(err)
		return
	}
	if vv := mustPassTests[idx].v.(myInt); vv != v11 {
		t.Error("got different values", v11, vv)
		return
	}

	idx++
	b, _ = hex.DecodeString(mustPassTests[idx].expHex)
	var v12 bool
	if _, err := Unmarshal(b, &v12); err != nil {
		t.Error(err)
		return
	}
	if mustPassTests[idx].v.(bool) != v12 {
		t.Error("got different values")
		return
	}

	idx++
	b, _ = hex.DecodeString(mustPassTests[idx].expHex)
	var v13 float32
	if _, err := Unmarshal(b, &v13); err != nil {
		t.Error(err)
		return
	}
	if mustPassTests[idx].v.(float32) != v13 {
		t.Error("got different values")
		return
	}

	idx++
	b, _ = hex.DecodeString(mustPassTests[idx].expHex)
	var v14 float64
	if _, err := Unmarshal(b, &v14); err != nil {
		t.Error(err)
		return
	}
	if mustPassTests[idx].v.(float64) != v14 {
		t.Error("got different values")
		return
	}

	idx++
	b, _ = hex.DecodeString(mustPassTests[idx].expHex)
	var v15 complex64
	if _, err := Unmarshal(b, &v15); err != nil {
		t.Error(err)
		return
	}
	if mustPassTests[idx].v.(complex64) != v15 {
		t.Error("got different values")
		return
	}

	idx++
	b, _ = hex.DecodeString(mustPassTests[idx].expHex)
	var v16 complex128
	if _, err := Unmarshal(b, &v16); err != nil {
		t.Error(err)
		return
	}
	if mustPassTests[idx].v.(complex128) != v16 {
		t.Error("got different values")
		return
	}

	idx++
	b, _ = hex.DecodeString(mustPassTests[idx].expHex)
	var v17 string
	if _, err := Unmarshal(b, &v17); err != nil {
		t.Error(err)
		return
	}
	if mustPassTests[idx].v.(string) != v17 {
		t.Error("got different values")
		return
	}

	idx += 2
	b, _ = hex.DecodeString(mustPassTests[idx].expHex)
	var v18 MyStruct = MyStruct{}
	if _, err := Unmarshal(b, &v18); err != nil {
		t.Error(err)
		return
	}
	if vv := *mustPassTests[idx].v.(*MyStruct); vv != v18 && *vv.C != *v18.C {
		t.Error("got different values")
		t.Log(vv)
		t.Log(v18)
		return
	}

	idx++
	b, _ = hex.DecodeString(mustPassTests[idx].expHex)
	var v19 [5]uint16
	if _, err := Unmarshal(b, &v19); err != nil {
		t.Error(err)
		return
	}
	if mustPassTests[idx].v.([5]uint16) != v19 {
		t.Error("got different values")
		return
	}

	idx++
	b, _ = hex.DecodeString(mustPassTests[idx].expHex)
	var v20 []uint16
	if _, err := Unmarshal(b, &v20); err != nil {
		t.Error(err)
		return
	}
	expSlice := mustPassTests[idx].v.([]uint16)
	if len(v20) != len(expSlice) {
		t.Error("got different values", len(v20), len(expSlice))
		return
	}
	for i, vv := range expSlice {
		if vv != v20[i] {
			t.Error("got different values")
			return
		}
	}

	idx++
	b, _ = hex.DecodeString(mustPassTests[idx].expHex)
	var v21 map[string]uint16
	if _, err := Unmarshal(b, &v21); err != nil {
		t.Error(err)
		return
	}
	expMap := mustPassTests[idx].v.(map[string]uint16)
	if len(expMap) != len(v21) {
		t.Error("got different values")
		return
	}
	for k, v := range expMap {
		if v21[k] != v {
			t.Error("got different values")
			return
		}
	}
}
