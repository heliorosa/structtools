package structtools

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"reflect"
)

var (
	// not a data struct
	ErrNotAStruct = errors.New("Not a struct")
	// not a pointer to a struct
	ErrNotAStructPtr = errors.New("Not a pointer to a struct")
	// type mismatch
	ErrDataTypesDontMatch = errors.New("data types don't match")
	// can't set value
	ErrCantSet = errors.New("can't set field")
)

// FromMap sets the fields of the struct s that are tagged with t
// and exist in m. The fields that are looked up and set, depend on
// the onlyTagged value. If is set to true, FromMap only looks
// for fields with a tag, otherwise it attempts to set every field.
func FromMap(t string, m map[string]interface{}, s interface{}, onlyTagged bool) error {
	// we need a pointer to a struct
	if reflect.TypeOf(s).Kind() != reflect.Ptr {
		return ErrNotAStructPtr
	}
	vals := reflect.ValueOf(s).Elem()
	typ := reflect.TypeOf(s).Elem()
	if vals.Type().Kind() != reflect.Struct {
		return ErrNotAStructPtr
	}
	// find values in the map with a corresponding name
	for i := 0; i < vals.NumField(); i++ {
		fldTyp := typ.Field(i)
		tagVal := fldTyp.Tag.Get(t)
		// if the tag is "-", ignore the field and continue
		if tagVal == "-" {
			continue
		}
		// if only tagged and no tag
		if onlyTagged && tagVal == "" {
			continue
		}
		// if the tag is set use it otherwise use the field name
		var fldName string
		if tagVal != "" {
			fldName = tagVal
		} else {
			fldName = fldTyp.Name
		}
		// check if it exists in the map
		mval, ok := m[fldName]
		if !ok {
			continue
		}
		// get the field
		v := vals.Field(i)
		// can we set the field ?
		if !v.CanSet() {
			return ErrCantSet
		}
		// check if the types are the same
		tmp := reflect.ValueOf(mval)
		if v.Type() != tmp.Type() {
			return ErrDataTypesDontMatch
		}
		v.Set(tmp)
	}
	return nil
}

// AddToMap adds the values in the fields of the struct s that are tagged with t to m
// AddToMap looks for the tag t on the field and uses it as a key in m, if there is no tag,
// the field name is used instead. Fields tagged with "-" are ignored and if onlyTagged
// is set to true, all fields are copied to m, otherwise, copy only occurs in tagged fields.
func AddToMap(t string, s interface{}, m map[string]interface{}, onlyTagged bool) error {
	vals := reflect.ValueOf(s)
	typ := reflect.TypeOf(s)
	if typ.Kind() == reflect.Ptr {
		vals = vals.Elem()
		typ = typ.Elem()
	}
	if vals.Type().Kind() != reflect.Struct {
		return ErrNotAStruct
	}
	for i := 0; i < vals.NumField(); i++ {
		fldTyp := typ.Field(i)
		tag := fldTyp.Tag.Get(t)
		if tag == "-" {
			continue
		}
		if onlyTagged && tag == "" {
			continue
		}
		var fldName string
		if tag == "" {
			fldName = fldTyp.Name
		} else {
			fldName = tag
		}
		m[fldName] = vals.Field(i).Interface()
	}
	return nil
}

// ToMap creates a new map with the fields of s that are tagged with t.
// This function follows the same rules defined for AddToMap().
func ToMap(t string, s interface{}, onlyTagged bool) (map[string]interface{}, error) {
	m := make(map[string]interface{})
	err := AddToMap(t, s, m, onlyTagged)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// Marshaler should be implemented by any types with custom marshaling
type Marshaler interface {
	MarshalBinary(w io.Writer) (int, error)
}

// Unmarshaler should be implemented by any types with custom unmarshalling
type Unmarshaler interface {
	UnmarshalBinary(r io.Reader) (int, error)
}

// DefaultTag that is looked up when the field OnlyTagged
// of the Encoder and Decoder types are set to true.
const DefaultTag = "bin"

// DefaultByteOrder is the default byte order for the
// ByteOrder field of the Encoder/Decoder.
var DefaultByteOrder = binary.BigEndian

// Marshal value v
func Marshal(v interface{}) ([]byte, error) {
	b := bytes.NewBuffer(make([]byte, 0, 128))
	if err := NewEncoder(b).Encode(v); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func MarshalOnly(v interface{}, tag string) ([]byte, error) {
	b := bytes.NewBuffer(make([]byte, 0, 128))
	enc := NewEncoder(b)
	enc.OnlyTagged = true
	enc.Tag = tag
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// Encoder is used to marshal several values to an io.Writer
type Encoder struct {
	w io.Writer
	// byte order
	ByteOrder binary.ByteOrder
	// tag to look for
	Tag string
	// only marshal tagged fields
	OnlyTagged bool
}

// NewEncoder creates a new encoder that writes to w. The field DefaultTag
// defaults to DefaultTag and ByteOrder to binary.BigEndian
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w: w, Tag: DefaultTag, ByteOrder: DefaultByteOrder}
}

// Encode value V
func (e *Encoder) Encode(v interface{}) error { return encode(e, v) }

// will not marshal/unmarshal those
var forbiddenKinds = []reflect.Kind{
	reflect.Invalid,
	reflect.Uintptr,
	reflect.UnsafePointer,
	reflect.Chan,
	reflect.Func,
	reflect.Interface,
}

func isForbiddenKind(k reflect.Kind) reflect.Kind { return inSlice(k, forbiddenKinds) }
func inSlice(k reflect.Kind, s []reflect.Kind) reflect.Kind {
	for _, ik := range s {
		if ik == k {
			return k
		}
	}
	return reflect.Invalid

}

func writeAll(w io.Writer, b []byte) error {
	sz, err := io.Copy(w, bytes.NewReader(b))
	if err != nil {
		return err
	}
	if sz != int64(len(b)) {
		return fmt.Errorf("only %d bytes of %d written", sz, len(b))
	}
	return nil
}

func encode(enc *Encoder, v interface{}) error {
	// don't handle forbidden kinds
	val := reflect.ValueOf(v)
	if k := isForbiddenKind(val.Kind()); k != reflect.Invalid {
		return fmt.Errorf("can't handle %s", k.String())
	}

	// got a marshaler
	if marshaler, ok := v.(Marshaler); ok {
		_, err := marshaler.MarshalBinary(enc.w)
		return err
	}

	// got a pointer
	if k := val.Kind(); k == reflect.Invalid {
		return nil
	} else if k == reflect.Ptr {
		// anything to marshal
		if val.IsNil() {
			return nil
		}
		val = val.Elem()
	}

	var b []byte
	switch v := val.Interface().(type) {
	// ints
	case int8:
		b = []byte{byte(v)}
	case uint8:
		b = []byte{byte(v)}
	case uint16:
		b = make([]byte, 2)
		enc.ByteOrder.PutUint16(b, v)
	case uint32:
		b = make([]byte, 4)
		enc.ByteOrder.PutUint32(b, v)
	case uint64:
		b = make([]byte, 8)
		enc.ByteOrder.PutUint64(b, v)
	case uint:
		b = make([]byte, 8)
		enc.ByteOrder.PutUint64(b, uint64(v))
	case int16:
		b = make([]byte, 2)
		enc.ByteOrder.PutUint16(b, uint16(v))
	case int32:
		b = make([]byte, 4)
		enc.ByteOrder.PutUint32(b, uint32(v))
	case int64:
		b = make([]byte, 8)
		enc.ByteOrder.PutUint64(b, uint64(v))
	case int:
		b = make([]byte, 8)
		enc.ByteOrder.PutUint64(b, uint64(v))
	// floats
	case float32:
		b = make([]byte, 4)
		enc.ByteOrder.PutUint32(b, math.Float32bits(v))
	case float64:
		b = make([]byte, 8)
		enc.ByteOrder.PutUint64(b, math.Float64bits(v))
	// complexes
	case complex64:
		b = make([]byte, 8)
		enc.ByteOrder.PutUint32(b, math.Float32bits(real(v)))
		enc.ByteOrder.PutUint32(b[4:], math.Float32bits(imag(v)))
	case complex128:
		b = make([]byte, 16)
		enc.ByteOrder.PutUint64(b, math.Float64bits(real(v)))
		enc.ByteOrder.PutUint64(b[8:], math.Float64bits(imag(v)))
	// bools
	case bool:
		var bb byte
		if v {
			bb = 1
		}
		b = []byte{bb}
	// strings
	case string:
		b = []byte(v)
		if err := encode(enc, uint64(len(b))); err != nil {
			return err
		}
	default:
		switch k := val.Kind(); k {
		// structs
		case reflect.Struct:
			for i := 0; i < val.NumField(); i++ {
				fldTyp := val.Type().Field(i)
				if tag := fldTyp.Tag.Get(enc.Tag); enc.OnlyTagged && (tag == "" || tag == "-") {
					continue
				}
				if err := encode(enc, val.Field(i).Interface()); err != nil {
					return err
				}
			}
			return nil
		// arrays and slices
		case reflect.Array, reflect.Slice:
			if k == reflect.Slice {
				if err := encode(enc, uint64(val.Len())); err != nil {
					return err
				}
			}
			for i := 0; i < val.Len(); i++ {
				if err := encode(enc, val.Index(i).Interface()); err != nil {
					return err
				}
			}
			return nil
		// maps
		case reflect.Map:
			ve := val.Type().Elem().Kind()
			vk := val.Type().Key().Kind()
			if vk == reflect.Interface || ve == reflect.Interface {
				return fmt.Errorf("will not encode a map with interface{} as keys/values")
			}
			if err := encode(enc, uint64(val.Len())); err != nil {
				return err
			}
			for _, k := range val.MapKeys() {
				if err := encode(enc, k.Interface()); err != nil {
					return err
				}
				if err := encode(enc, val.MapIndex(k).Interface()); err != nil {
					return err
				}
			}
			return nil
		}
	}
	return writeAll(enc.w, b)
}

// Decoder can be used to unmarshal several values from an io.Reader
type Decoder struct {
	r io.Reader
	// byte order
	ByteOrder binary.ByteOrder
	// tag to look for
	Tag string
	// only unmarshal tagged fields
	OnlyTagged bool
}

// NewDecoder creates a new decoder that reads from r
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r: r, Tag: DefaultTag, ByteOrder: DefaultByteOrder}
}

// Decode value v
func (d *Decoder) Decode(v interface{}) error { return decode(d, v) }

// Unmarshal data into v and return number of used bytes or an error
func Unmarshal(data []byte, v interface{}) (int, error) {
	b := bytes.NewReader(data)
	if err := NewDecoder(b).Decode(v); err != nil {
		return 0, err
	}
	return len(data) - b.Len(), nil
}

func UnmarshalOnly(data []byte, v interface{}, tag string) (int, error) {
	b := bytes.NewReader(data)
	dec := NewDecoder(b)
	dec.OnlyTagged = true
	dec.Tag = tag
	if err := dec.Decode(v); err != nil {
		return 0, err
	}
	return len(data) - b.Len(), nil
}

func readN(r io.Reader, n uint64) ([]byte, error) {
	b := make([]byte, n)
	sz := uint64(0)
	for sz < n {
		nBytes, err := r.Read(b[sz:])
		if err != nil {
			if err != io.EOF {
				return b[:sz], err
			}
		}
		if nBytes == 0 {
			break
		}
		sz += uint64(nBytes)
	}
	return b, nil
}

func decode(dec *Decoder, v interface{}) error {
	// don't handle forbidden kinds
	val := reflect.ValueOf(v)
	if k := isForbiddenKind(val.Kind()); k != reflect.Invalid {
		return fmt.Errorf("can't handle %s", k.String())
	}

	// got a unmarshaler
	if unmarshaler, ok := v.(Unmarshaler); ok {
		_, err := unmarshaler.UnmarshalBinary(dec.r)
		if err != nil {
			return err
		}
		return nil
	}

	// check if it's a pointer not nil
	if k := val.Kind(); k != reflect.Ptr {
		return fmt.Errorf("can only unmarshal to a pointer")
	} else if val.IsNil() {
		return nil
	}
	val = val.Elem()

	switch val.Interface().(type) {
	// ints
	case uint8:
		b, err := readN(dec.r, 1)
		if err != nil {
			return err
		}
		val.SetUint(uint64(b[0]))
	case uint16:
		b, err := readN(dec.r, 2)
		if err != nil {
			return err
		}
		val.SetUint(uint64(dec.ByteOrder.Uint16(b)))
	case uint32:
		b, err := readN(dec.r, 4)
		if err != nil {
			return err
		}
		val.SetUint(uint64(dec.ByteOrder.Uint32(b)))
	case uint64:
		b, err := readN(dec.r, 8)
		if err != nil {
			return err
		}
		val.SetUint(dec.ByteOrder.Uint64(b))
	case uint:
		b, err := readN(dec.r, 8)
		if err != nil {
			return err
		}
		val.SetUint(dec.ByteOrder.Uint64(b))
	case int8:
		b, err := readN(dec.r, 1)
		if err != nil {
			return err
		}
		val.SetInt(int64(b[0]))
	case int16:
		b, err := readN(dec.r, 2)
		if err != nil {
			return err
		}
		val.SetInt(int64(dec.ByteOrder.Uint16(b)))
	case int32:
		b, err := readN(dec.r, 4)
		if err != nil {
			return err
		}
		val.SetInt(int64(dec.ByteOrder.Uint32(b)))
	case int64:
		b, err := readN(dec.r, 8)
		if err != nil {
			return err
		}
		val.SetInt(int64(dec.ByteOrder.Uint64(b)))
	case int:
		b, err := readN(dec.r, 8)
		if err != nil {
			return err
		}
		val.SetInt(int64(dec.ByteOrder.Uint64(b)))
	// floats
	case float32:
		b, err := readN(dec.r, 4)
		if err != nil {
			return err
		}
		val.SetFloat(float64(math.Float32frombits(dec.ByteOrder.Uint32(b))))
	case float64:
		b, err := readN(dec.r, 8)
		if err != nil {
			return err
		}
		val.SetFloat(math.Float64frombits(dec.ByteOrder.Uint64(b)))
	// complexes
	case complex64:
		b, err := readN(dec.r, 8)
		if err != nil {
			return err
		}
		val.SetComplex(complex128(complex(
			math.Float32frombits(dec.ByteOrder.Uint32(b)),
			math.Float32frombits(dec.ByteOrder.Uint32(b[4:])),
		)))
	case complex128:
		b, err := readN(dec.r, 16)
		if err != nil {
			return err
		}
		val.SetComplex(complex(
			math.Float64frombits(dec.ByteOrder.Uint64(b)),
			math.Float64frombits(dec.ByteOrder.Uint64(b[8:])),
		))
	// bools
	case bool:
		b, err := readN(dec.r, 1)
		if err != nil {
			return err
		}
		var v bool
		if b[0] != 0 {
			v = true
		}
		val.SetBool(v)
	// strings
	case string:
		var sz uint64
		if err := decode(dec, &sz); err != nil {
			return err
		}
		b, err := readN(dec.r, sz)
		if err != nil {
			return err
		}
		val.SetString(string(b))
	default:
		switch k := val.Kind(); k {
		// structs
		case reflect.Struct:
			for i := 0; i < val.NumField(); i++ {
				fldTyp := val.Type().Field(i)
				if tag := fldTyp.Tag.Get(dec.Tag); dec.OnlyTagged && (tag == "" || tag == "-") {
					continue
				}
				fldVal := val.Field(i)
				if fldVal.Kind() != reflect.Ptr {
					fldVal = fldVal.Addr()
				} else if fldVal.IsNil() {
					newV := reflect.New(fldTyp.Type.Elem())
					fldVal.Set(newV)
				}
				if err := decode(dec, fldVal.Interface()); err != nil {
					return err
				}
			}
		// arrays and slices
		case reflect.Array, reflect.Slice:
			var (
				addElem func(int, reflect.Value)
				sz      uint64
			)
			if k == reflect.Slice {
				newS := reflect.New(reflect.SliceOf(val.Type().Elem()))
				val.Set(newS.Elem())
				if err := decode(dec, &sz); err != nil {
					return err
				}
				addElem = func(i int, v reflect.Value) { val.Set(reflect.Append(val, v)) }
			} else {
				sz = uint64(val.Len())
				addElem = func(i int, v reflect.Value) { val.Index(i).Set(v) }
			}

			for i := uint64(0); i < sz; i++ {
				v := reflect.New(val.Type().Elem())
				if err := decode(dec, v.Interface()); err != nil {
					return err
				}
				addElem(int(i), v.Elem())
			}
		// maps
		case reflect.Map:
			if vk, ve := val.Type().Key().Kind(), val.Type().Elem().Kind(); ve == reflect.Interface || vk == reflect.Interface {
				return fmt.Errorf("will not encode a map with interface{} as key/value")
			}
			var sz uint64
			if err := decode(dec, &sz); err != nil {
				return err
			}
			val.Set(reflect.MakeMap(val.Type()))
			for i := uint64(0); i < sz; i++ {
				k := reflect.New(val.Type().Key())
				v := reflect.New(val.Type().Elem())
				if err := decode(dec, k.Interface()); err != nil {
					return err
				}
				if err := decode(dec, v.Interface()); err != nil {
					return err
				}
				val.SetMapIndex(k.Elem(), v.Elem())
			}
		}

	}
	return nil
}
