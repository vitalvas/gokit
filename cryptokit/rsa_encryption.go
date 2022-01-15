package cryptokit

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha512"
	"hash"
)

type RSAEncryption struct {
	hash hash.Hash
}

func NewRSAEncryption() *RSAEncryption {
	return &RSAEncryption{
		hash: sha512.New(),
	}
}

func (e *RSAEncryption) SetHash(h hash.Hash) {
	e.hash = h
}

func (e *RSAEncryption) EncryptWithPublicKey(msg []byte, pub *rsa.PublicKey) ([]byte, error) {
	ciphertext, err := rsa.EncryptOAEP(e.hash, rand.Reader, pub, msg, nil)
	if err != nil {
		return nil, err
	}

	return ciphertext, nil
}

func (e *RSAEncryption) DecryptWithPrivateKey(ciphertext []byte, priv *rsa.PrivateKey) ([]byte, error) {
	plaintext, err := rsa.DecryptOAEP(e.hash, rand.Reader, priv, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}
