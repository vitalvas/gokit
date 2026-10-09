package secp256k1

import "errors"

var errInvalidScalar = errors.New("secp256k1: invalid scalar")

// scalarValue stores a scalar modulo N in Montgomery form.
type scalarValue struct {
	v uint256
}

func newScalarValue() *scalarValue { return &scalarValue{} }

func (s *scalarValue) decode(raw []byte) error {
	if len(raw) != 32 {
		return errInvalidScalar
	}
	v := decodeWords(raw)
	if _, borrow := subtractWords(v, scalarModulus.n); borrow == 0 {
		return errInvalidScalar
	}
	s.v = scalarModulus.toMontgomery(v)
	return nil
}

func (s *scalarValue) decodeReduced(raw []byte) error {
	if len(raw) != 32 {
		return errInvalidScalar
	}
	v := decodeWords(raw)
	reduced, borrow := subtractWords(v, scalarModulus.n)
	v = selectWords(v, reduced, borrow^1)
	s.v = scalarModulus.toMontgomery(v)
	return nil
}

func (s *scalarValue) encode() []byte {
	return encodeWords(scalarModulus.fromMontgomery(s.v))
}

func (s *scalarValue) isZero() bool { return zeroBit(s.v) == 1 }

func (s *scalarValue) copy() *scalarValue { return &scalarValue{v: s.v} }

func (s *scalarValue) add(t *scalarValue) *scalarValue {
	s.v = addMod(s.v, t.v, scalarModulus)
	return s
}

func (s *scalarValue) subtract(t *scalarValue) *scalarValue {
	s.v = subtractMod(s.v, t.v, scalarModulus)
	return s
}

func (s *scalarValue) multiply(t *scalarValue) *scalarValue {
	s.v = montgomeryMultiply(s.v, t.v, scalarModulus)
	return s
}

func (s *scalarValue) invert() *scalarValue {
	s.v = scalarModulus.inverse(s.v)
	return s
}

func (s *scalarValue) lessOrEqual(t *scalarValue) uint64 {
	a := scalarModulus.fromMontgomery(s.v)
	b := scalarModulus.fromMontgomery(t.v)
	_, borrow := subtractWords(b, a)
	return borrow ^ 1
}

// Choice 0 selects u; choice 1 selects v.
func (s *scalarValue) selectValue(choice uint64, u, v *scalarValue) error {
	if choice > 1 || u == nil || v == nil {
		return errInvalidScalar
	}
	s.v = selectWords(u.v, v.v, choice)
	return nil
}
