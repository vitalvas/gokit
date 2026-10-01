package secp256k1

// ECDH computes the Elliptic Curve Diffie-Hellman shared secret between priv and
// the peer public key pub, returning the 32-byte big-endian X coordinate of the
// shared point priv.D * pub.
//
// As is standard for ECDH, the raw X coordinate is returned without hashing;
// callers must run it through a KDF (e.g. HKDF or a plain hash) before using it
// as key material. The peer key is validated to be on the curve.
func (p *PrivateKey) ECDH(pub *PublicKey) ([]byte, error) {
	if p == nil || p.D == nil {
		return nil, errBadKey("nil private key")
	}

	if !pub.IsValid() {
		return nil, errBadKey("peer public key is not on the curve")
	}

	shared := scalarMult(affineToJacobian(pub.X, pub.Y), p.D)
	x, _ := shared.toAffine()
	if shared.isIdentity() {
		return nil, errBadKey("shared secret is the point at infinity")
	}

	return fixedBytes(x), nil
}
