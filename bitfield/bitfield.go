package bitfield

import (
	"fmt"
	"log"

	"gopkg.in/mgo.v2/bson"
)

type BitField struct {
	Fields []byte
}

func New(size int) *BitField {
	// Bit field of size 0 is meaningless. Return nil.
	if size == 0 {
		panic("seeking a zero-size bitfield")
	}

	// 8 bits per byte.
	numbytes := (size-1)/8 + 1

	f := &BitField{
		Fields: make([]byte, numbytes, numbytes),
	}

	return f
}

func (f BitField) String() string {
	return fmt.Sprintf("%x", f.Fields)
}

func (f BitField) length() int {
	return len(f.Fields)
}

func (f BitField) locate(pos int) (byt int, bit uint8) {
	byt = pos >> 3
	bit = uint8(pos & 0x7)
	return
}

func posof(byt int, bit uint8) int {
	return (byt << 3) | int(bit)
}

func (f *BitField) Set(positions ...int) {
	for _, pos := range positions {
		byt, bit := f.locate(pos)
		if byt > f.length()-1 {
			log.Panicf("BitField can set max pos %d. Attempting %d.",
				f.length()*8-1, pos)
		}
		f.Fields[byt] |= (1 << bit)
	}
}

func (f *BitField) SetBitsOf(b BitField) {
	if f.length() != b.length() {
		log.Panic("SetBitsOf: mismatch in lengths")
	}
	for i := range f.Fields {
		f.Fields[i] |= b.Fields[i]
	}
}

func (f *BitField) Unset(positions ...int) {
	for _, pos := range positions {
		byt, bit := f.locate(pos)
		if byt > f.length()-1 {
			log.Panicf("BitField can unset max pos %d. Attempting %d.",
				f.length()*8-1, pos)
		}
		f.Fields[byt] &= ^(1 << bit)
	}
}

func (f BitField) Test() (setpositions []int) {
	for i := range f.Fields {
		for j := uint8(0); j < 8; j++ {
			mask := uint8(1) << j
			if f.Fields[i]&mask != 0 {
				setpositions = append(setpositions, posof(i, j))
			}
		}
	}
	return
}

func (f BitField) AllUnset() bool {
	var acc byte
	for _, b := range f.Fields {
		acc |= b
	}
	return acc == 0
}

// GetBSON makes BitField implement bson.Getter
func (f BitField) GetBSON() (interface{}, error) {
    // We want to save the bitfield as a simple hexadecimal string
	return f.String(), nil
}

// SetBSON makes BitField implement bson.Setter
func (f *BitField) SetBSON(raw bson.Raw) error {
	var str string
    // We need to unmarshal the raw data into a string before it can be
    // interpreted. It will unmarshal into a string because we encoded it as a
    // string in the Getter - GetBSON.
	err := raw.Unmarshal(&str)
	if err != nil {
		return err
	}
	_, err = fmt.Sscanf(str, "%x", &f.Fields) // Opposite of what BitField.String() does
    if err != nil {
        return err
    }
    return nil
}
