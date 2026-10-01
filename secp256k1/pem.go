package secp256k1

import (
	"encoding/asn1"
	"encoding/pem"
)

// oidSecp256k1 is the ASN.1 object identifier for the secp256k1 curve
// (SEC 2, 1.3.132.0.10). crypto/x509 does not know this curve, so this package
// marshals keys with the standard RFC 5915 / RFC 5480 structures directly.
var oidSecp256k1 = asn1.ObjectIdentifier{1, 3, 132, 0, 10}

// ecPrivateKey mirrors RFC 5915 ECPrivateKey with the named-curve parameters
// and public key carried in their context-tagged optional fields.
type ecPrivateKey struct {
	Version       int
	PrivateKey    []byte
	NamedCurveOID asn1.ObjectIdentifier `asn1:"optional,explicit,tag:0"`
	PublicKey     asn1.BitString        `asn1:"optional,explicit,tag:1"`
}

// pkixPublicKey mirrors RFC 5480 SubjectPublicKeyInfo for an EC key.
type pkixPublicKey struct {
	Algorithm pkixAlgorithm
	PublicKey asn1.BitString
}

type pkixAlgorithm struct {
	Algorithm  asn1.ObjectIdentifier
	Parameters asn1.ObjectIdentifier
}

// oidPublicKeyECDSA is id-ecPublicKey (1.2.840.10045.2.1).
var oidPublicKeyECDSA = asn1.ObjectIdentifier{1, 2, 840, 10045, 2, 1}

// MarshalECPrivateKey encodes priv in the RFC 5915 SEC1 DER form (the body of a
// "EC PRIVATE KEY" PEM block).
func MarshalECPrivateKey(priv *PrivateKey) ([]byte, error) {
	return asn1.Marshal(ecPrivateKey{
		Version:       1,
		PrivateKey:    priv.Serialize(),
		NamedCurveOID: oidSecp256k1,
		PublicKey:     asn1.BitString{Bytes: priv.Pub.SerializeUncompressed(), BitLength: 8 * pubKeyUncompressedLen},
	})
}

// ParseECPrivateKey decodes an RFC 5915 SEC1 DER private key for secp256k1.
func ParseECPrivateKey(der []byte) (*PrivateKey, error) {
	var key ecPrivateKey

	rest, err := asn1.Unmarshal(der, &key)
	if err != nil {
		return nil, errBadKey("invalid EC private key DER: %v", err)
	}

	if len(rest) != 0 {
		return nil, errBadKey("trailing data after EC private key")
	}

	if len(key.NamedCurveOID) != 0 && !key.NamedCurveOID.Equal(oidSecp256k1) {
		return nil, errBadKey("unexpected curve OID %v, want secp256k1", key.NamedCurveOID)
	}

	return PrivKeyFromBytes(key.PrivateKey)
}

// MarshalPKIXPublicKey encodes pub in RFC 5480 SubjectPublicKeyInfo DER form
// (the body of a "PUBLIC KEY" PEM block), using the uncompressed point.
func MarshalPKIXPublicKey(pub *PublicKey) ([]byte, error) {
	return asn1.Marshal(pkixPublicKey{
		Algorithm: pkixAlgorithm{Algorithm: oidPublicKeyECDSA, Parameters: oidSecp256k1},
		PublicKey: asn1.BitString{Bytes: pub.SerializeUncompressed(), BitLength: 8 * pubKeyUncompressedLen},
	})
}

// ParsePKIXPublicKey decodes an RFC 5480 SubjectPublicKeyInfo DER public key for
// secp256k1.
func ParsePKIXPublicKey(der []byte) (*PublicKey, error) {
	var spki pkixPublicKey

	rest, err := asn1.Unmarshal(der, &spki)
	if err != nil {
		return nil, errBadKey("invalid public key DER: %v", err)
	}

	if len(rest) != 0 {
		return nil, errBadKey("trailing data after public key")
	}

	if !spki.Algorithm.Algorithm.Equal(oidPublicKeyECDSA) {
		return nil, errBadKey("not an EC public key")
	}

	if !spki.Algorithm.Parameters.Equal(oidSecp256k1) {
		return nil, errBadKey("unexpected curve OID %v, want secp256k1", spki.Algorithm.Parameters)
	}

	return ParsePubKey(spki.PublicKey.Bytes)
}

// MarshalPrivateKeyPEM encodes priv as a PEM "EC PRIVATE KEY" block.
func MarshalPrivateKeyPEM(priv *PrivateKey) ([]byte, error) {
	der, err := MarshalECPrivateKey(priv)
	if err != nil {
		return nil, err
	}

	return pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der}), nil
}

// MarshalPublicKeyPEM encodes pub as a PEM "PUBLIC KEY" block.
func MarshalPublicKeyPEM(pub *PublicKey) ([]byte, error) {
	der, err := MarshalPKIXPublicKey(pub)
	if err != nil {
		return nil, err
	}

	return pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}), nil
}

// ParsePrivateKeyPEM decodes a PEM "EC PRIVATE KEY" block.
func ParsePrivateKeyPEM(data []byte) (*PrivateKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errBadKey("no PEM block found")
	}

	return ParseECPrivateKey(block.Bytes)
}

// ParsePublicKeyPEM decodes a PEM "PUBLIC KEY" block.
func ParsePublicKeyPEM(data []byte) (*PublicKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errBadKey("no PEM block found")
	}

	return ParsePKIXPublicKey(block.Bytes)
}
