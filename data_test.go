package tiff66

import (
	"encoding/binary"
	"math"
	"testing"
)

// Test the get/put functions.
func doOrder(t *testing.T, order binary.ByteOrder) {
	var ifd IFD_T
	ifd.Fields = make([]Field, 1)
	ifd.Fields[0] = Field{Compression, BYTE, 1, nil}
	field := &ifd.Fields[0]
	field.Data = make([]byte, 16)
	pos := uint32(1)
	{
		val := uint8(42)
		field.PutByte(val, pos)
		if field.Byte(pos) != val {
			t.Error("Byte")
		}
	}
	{
		val := uint16(42)
		field.PutShort(val, pos, order)
		if field.Short(pos, order) != val {
			t.Error("Short")
		}
	}
	{
		val := uint32(42)
		field.PutLong(val, pos, order)
		if field.Long(pos, order) != val {
			t.Error("Long")
		}
	}
	{
		val := int8(-42)
		field.PutSByte(val, pos)
		if field.SByte(pos) != val {
			t.Error("SByte")
		}
	}
	{
		val := int16(-42)
		field.PutSShort(val, pos, order)
		if field.SShort(pos, order) != val {
			t.Error("SShort")
		}
	}
	{
		val := int32(-42)
		field.PutSLong(val, pos, order)
		if field.SLong(pos, order) != val {
			t.Error("SLong")
		}
	}
	{
		n := uint32(21)
		d := uint32(42)
		field.PutRational(n, d, pos, order)
		nr, dr := field.Rational(pos, order)
		if nr != n || dr != d {
			t.Error("Rational")
		}
	}
	{
		n := int32(-21)
		d := int32(-42)
		field.PutSRational(n, d, pos, order)
		nr, dr := field.SRational(pos, order)
		if nr != n || dr != d {
			t.Error("SRational")
		}
	}
	{
		val := float32(math.Pi)
		field.PutFloat(val, pos, order)
		if field.Float(pos, order) != val {
			t.Error("Float")
		}
	}
	{
		val := float64(math.Pi)
		field.PutDouble(val, pos, order)
		if field.Double(pos, order) != val {
			t.Error("Double")
		}
	}
	{
		val := "42"
		field.PutASCII(val)
		if field.ASCII() != val {
			t.Error("ASCII")
		}
	}
}

func TestData(t *testing.T) {
	doOrder(t, binary.BigEndian)
	doOrder(t, binary.LittleEndian)
}
