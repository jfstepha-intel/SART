package bitfield

import "log"

type BitField struct {
	fields []byte
}

func New(size int) *BitField {
	// Bit field of size 0 is meaningless. Return nil.
	if size == 0 {
		return nil
	}

	// 8 bits per byte.
	numbytes := (size-1)/8 + 1

	f := &BitField{
		fields: make([]byte, numbytes, numbytes),
	}

	return f
}

func (f BitField) length() int {
	return len(f.fields)
}

func (f BitField) locate(pos int) (byt int, bit uint8) {
	byt = pos >> 3
	bit = uint8(pos & 0x7)
	return
}

func (f *BitField) Set(positions ...int) {
	for _, pos := range positions {
		byt, bit := f.locate(pos)
		if byt > f.length()-1 {
			log.Fatalf("BitField can set max pos %d. Attempting %d.",
				f.length()*8-1, pos)
		}
		f.fields[byt] |= (1 << bit)
	}
}

func (f *BitField) Unset(positions ...int) {
	for _, pos := range positions {
		byt, bit := f.locate(pos)
		if byt > f.length()-1 {
			log.Fatalf("BitField can unset max pos %d. Attempting %d.",
				f.length()*8-1, pos)
		}
		f.fields[byt] &= ^(1 << bit)
	}
}

func (f BitField) Test() (setpositions []int) {
	for i := range f.fields {
		for j := uint8(0); j < 8; j++ {
			mask := uint8(1) << j
			if f.fields[i]&mask != 0 {
				setpositions = append(setpositions, posof(i, j))
			}
		}
	}
	return
}

func posof(byt int, bit uint8) int {
	return (byt << 3) | int(bit)
}
