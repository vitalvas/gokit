package wirefilter

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"regexp"
	"sync"
)

// Binary encoding format:
//
//	Header: "WF" (2 bytes) + version (1 byte)
//	Body: recursive AST node encoding
//
// Node type tags:
const (
	nodeTypeBinary       byte = 0x01
	nodeTypeUnary        byte = 0x02
	nodeTypeField        byte = 0x03
	nodeTypeLiteral      byte = 0x04
	nodeTypeArray        byte = 0x05
	nodeTypeRange        byte = 0x06
	nodeTypeIndex        byte = 0x07
	nodeTypeUnpack       byte = 0x08
	nodeTypeListRef      byte = 0x09
	nodeTypeFunctionCall byte = 0x0A
)

// Value type tags:
const (
	valTypeNil    byte = 0x00
	valTypeString byte = 0x01
	valTypeInt    byte = 0x02
	valTypeBool   byte = 0x03
	valTypeIP     byte = 0x04
	valTypeCIDR   byte = 0x05
	valTypeBytes  byte = 0x06
)

const (
	encodingMagic   = "WF"
	encodingVersion = 1
)

var (
	errInvalidMagic   = errors.New("wirefilter: invalid binary format")
	errInvalidVersion = errors.New("wirefilter: unsupported encoding version")
	errInvalidNode    = errors.New("wirefilter: invalid node type in binary data")
	errInvalidValue   = errors.New("wirefilter: invalid value type in binary data")
	errTruncated      = errors.New("wirefilter: truncated binary data")
)

// MarshalBinary encodes the compiled filter into a binary representation.
// The resulting bytes can be stored externally and later decoded with UnmarshalBinary
// to reconstruct the filter without re-parsing the expression.
func (f *Filter) MarshalBinary() ([]byte, error) {
	w := &encWriter{buf: make([]byte, 0, 256)}
	w.writeBytes([]byte(encodingMagic))
	w.writeByte(encodingVersion)

	if err := w.writeExpr(f.expr); err != nil {
		return nil, err
	}

	return w.buf, nil
}

// UnmarshalBinary reconstructs a compiled filter from binary data
// previously produced by MarshalBinary.
func (f *Filter) UnmarshalBinary(data []byte) error {
	if len(data) < 3 {
		return errInvalidMagic
	}
	if string(data[:2]) != encodingMagic {
		return errInvalidMagic
	}
	if data[2] != encodingVersion {
		return errInvalidVersion
	}

	r := &decReader{data: data, pos: 3}

	expr, err := r.readExpr()
	if err != nil {
		return err
	}

	f.expr = expr
	f.schema = nil
	f.regexCache = make(map[string]*regexp.Regexp)
	f.cidrCache = make(map[string]*net.IPNet)
	f.regexMu = sync.RWMutex{}
	f.cidrMu = sync.RWMutex{}

	return nil
}

// --- Encoder ---

type encWriter struct {
	buf []byte
}

func (w *encWriter) writeByte(b byte) {
	w.buf = append(w.buf, b)
}

func (w *encWriter) writeBytes(b []byte) {
	w.buf = append(w.buf, b...)
}

func (w *encWriter) writeUvarint(v uint64) {
	var tmp [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(tmp[:], v)
	w.buf = append(w.buf, tmp[:n]...)
}

func (w *encWriter) writeVarint(v int64) {
	var tmp [binary.MaxVarintLen64]byte
	n := binary.PutVarint(tmp[:], v)
	w.buf = append(w.buf, tmp[:n]...)
}

func (w *encWriter) writeString(s string) {
	w.writeUvarint(uint64(len(s)))
	w.buf = append(w.buf, s...)
}

func (w *encWriter) writeByteSlice(b []byte) {
	w.writeUvarint(uint64(len(b)))
	w.buf = append(w.buf, b...)
}

func (w *encWriter) writeExpr(expr Expression) error {
	switch e := expr.(type) {
	case *BinaryExpr:
		w.writeByte(nodeTypeBinary)
		w.writeByte(byte(e.Operator))
		if err := w.writeExpr(e.Left); err != nil {
			return err
		}
		return w.writeExpr(e.Right)

	case *UnaryExpr:
		w.writeByte(nodeTypeUnary)
		w.writeByte(byte(e.Operator))
		return w.writeExpr(e.Operand)

	case *FieldExpr:
		w.writeByte(nodeTypeField)
		w.writeString(e.Name)
		return nil

	case *LiteralExpr:
		w.writeByte(nodeTypeLiteral)
		return w.writeValue(e.Value)

	case *ArrayExpr:
		w.writeByte(nodeTypeArray)
		w.writeUvarint(uint64(len(e.Elements)))
		for _, elem := range e.Elements {
			if err := w.writeExpr(elem); err != nil {
				return err
			}
		}
		return nil

	case *RangeExpr:
		w.writeByte(nodeTypeRange)
		if err := w.writeExpr(e.Start); err != nil {
			return err
		}
		return w.writeExpr(e.End)

	case *IndexExpr:
		w.writeByte(nodeTypeIndex)
		if err := w.writeExpr(e.Object); err != nil {
			return err
		}
		return w.writeExpr(e.Index)

	case *UnpackExpr:
		w.writeByte(nodeTypeUnpack)
		return w.writeExpr(e.Array)

	case *ListRefExpr:
		w.writeByte(nodeTypeListRef)
		w.writeString(e.Name)
		return nil

	case *FunctionCallExpr:
		w.writeByte(nodeTypeFunctionCall)
		w.writeString(e.Name)
		w.writeUvarint(uint64(len(e.Arguments)))
		for _, arg := range e.Arguments {
			if err := w.writeExpr(arg); err != nil {
				return err
			}
		}
		return nil
	}

	return fmt.Errorf("wirefilter: unknown expression type: %T", expr)
}

func (w *encWriter) writeValue(v Value) error {
	if v == nil {
		w.writeByte(valTypeNil)
		return nil
	}

	switch val := v.(type) {
	case StringValue:
		w.writeByte(valTypeString)
		w.writeString(string(val))

	case IntValue:
		w.writeByte(valTypeInt)
		w.writeVarint(int64(val))

	case BoolValue:
		w.writeByte(valTypeBool)
		if val {
			w.writeByte(1)
		} else {
			w.writeByte(0)
		}

	case IPValue:
		w.writeByte(valTypeIP)
		ip := val.IP
		if ip4 := ip.To4(); ip4 != nil {
			w.writeByteSlice(ip4)
		} else {
			w.writeByteSlice(ip.To16())
		}

	case CIDRValue:
		w.writeByte(valTypeCIDR)
		w.writeByteSlice(val.IPNet.IP)
		w.writeByteSlice(val.IPNet.Mask)

	case BytesValue:
		w.writeByte(valTypeBytes)
		w.writeByteSlice([]byte(val))

	default:
		return fmt.Errorf("wirefilter: unknown value type: %T", v)
	}

	return nil
}

// --- Decoder ---

type decReader struct {
	data []byte
	pos  int
}

func (r *decReader) readByte() byte {
	if r.pos >= len(r.data) {
		return 0
	}
	b := r.data[r.pos]
	r.pos++
	return b
}

func (r *decReader) readN(n int) []byte {
	if r.pos+n > len(r.data) {
		return nil
	}
	b := r.data[r.pos : r.pos+n]
	r.pos += n
	return b
}

func (r *decReader) readUvarint() (uint64, error) {
	v, n := binary.Uvarint(r.data[r.pos:])
	if n <= 0 {
		return 0, errTruncated
	}
	r.pos += n
	return v, nil
}

func (r *decReader) readVarint() (int64, error) {
	v, n := binary.Varint(r.data[r.pos:])
	if n <= 0 {
		return 0, errTruncated
	}
	r.pos += n
	return v, nil
}

func (r *decReader) readString() (string, error) {
	length, err := r.readUvarint()
	if err != nil {
		return "", err
	}
	b := r.readN(int(length))
	if b == nil {
		return "", errTruncated
	}
	return string(b), nil
}

func (r *decReader) readByteSlice() ([]byte, error) {
	length, err := r.readUvarint()
	if err != nil {
		return nil, err
	}
	b := r.readN(int(length))
	if b == nil {
		return nil, errTruncated
	}
	dst := make([]byte, len(b))
	copy(dst, b)
	return dst, nil
}

func (r *decReader) eof() bool {
	return r.pos >= len(r.data)
}

func (r *decReader) readExpr() (Expression, error) {
	if r.eof() {
		return nil, io.ErrUnexpectedEOF
	}

	tag := r.readByte()

	switch tag {
	case nodeTypeBinary:
		op := TokenType(r.readByte())
		left, err := r.readExpr()
		if err != nil {
			return nil, err
		}
		right, err := r.readExpr()
		if err != nil {
			return nil, err
		}
		return &BinaryExpr{Left: left, Operator: op, Right: right}, nil

	case nodeTypeUnary:
		op := TokenType(r.readByte())
		operand, err := r.readExpr()
		if err != nil {
			return nil, err
		}
		return &UnaryExpr{Operator: op, Operand: operand}, nil

	case nodeTypeField:
		name, err := r.readString()
		if err != nil {
			return nil, err
		}
		return &FieldExpr{Name: name}, nil

	case nodeTypeLiteral:
		val, err := r.readValue()
		if err != nil {
			return nil, err
		}
		return &LiteralExpr{Value: val}, nil

	case nodeTypeArray:
		count, err := r.readUvarint()
		if err != nil {
			return nil, err
		}
		elements := make([]Expression, count)
		for i := range elements {
			elements[i], err = r.readExpr()
			if err != nil {
				return nil, err
			}
		}
		return &ArrayExpr{Elements: elements}, nil

	case nodeTypeRange:
		start, err := r.readExpr()
		if err != nil {
			return nil, err
		}
		end, err := r.readExpr()
		if err != nil {
			return nil, err
		}
		return &RangeExpr{Start: start, End: end}, nil

	case nodeTypeIndex:
		object, err := r.readExpr()
		if err != nil {
			return nil, err
		}
		index, err := r.readExpr()
		if err != nil {
			return nil, err
		}
		return &IndexExpr{Object: object, Index: index}, nil

	case nodeTypeUnpack:
		array, err := r.readExpr()
		if err != nil {
			return nil, err
		}
		return &UnpackExpr{Array: array}, nil

	case nodeTypeListRef:
		name, err := r.readString()
		if err != nil {
			return nil, err
		}
		return &ListRefExpr{Name: name}, nil

	case nodeTypeFunctionCall:
		name, err := r.readString()
		if err != nil {
			return nil, err
		}
		count, err := r.readUvarint()
		if err != nil {
			return nil, err
		}
		args := make([]Expression, count)
		for i := range args {
			args[i], err = r.readExpr()
			if err != nil {
				return nil, err
			}
		}
		return &FunctionCallExpr{Name: name, Arguments: args}, nil
	}

	return nil, fmt.Errorf("%w: 0x%02x", errInvalidNode, tag)
}

func (r *decReader) readValue() (Value, error) {
	if r.eof() {
		return nil, io.ErrUnexpectedEOF
	}

	tag := r.readByte()

	switch tag {
	case valTypeNil:
		return nil, nil

	case valTypeString:
		s, err := r.readString()
		if err != nil {
			return nil, err
		}
		return StringValue(s), nil

	case valTypeInt:
		v, err := r.readVarint()
		if err != nil {
			return nil, err
		}
		return IntValue(v), nil

	case valTypeBool:
		return BoolValue(r.readByte() != 0), nil

	case valTypeIP:
		b, err := r.readByteSlice()
		if err != nil {
			return nil, err
		}
		return IPValue{IP: net.IP(b)}, nil

	case valTypeCIDR:
		ipBytes, err := r.readByteSlice()
		if err != nil {
			return nil, err
		}
		maskBytes, err := r.readByteSlice()
		if err != nil {
			return nil, err
		}
		return CIDRValue{IPNet: &net.IPNet{IP: net.IP(ipBytes), Mask: net.IPMask(maskBytes)}}, nil

	case valTypeBytes:
		b, err := r.readByteSlice()
		if err != nil {
			return nil, err
		}
		return BytesValue(b), nil
	}

	return nil, fmt.Errorf("%w: 0x%02x", errInvalidValue, tag)
}
