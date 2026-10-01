package secp256k1

import "math/big"

// SignRecoverable signs hash under priv and returns the canonical low-S
// signature together with a recovery id in [0,3]. The recovery id lets a
// verifier reconstruct the signing public key from the signature alone via
// RecoverPubKey -- the mechanism behind Ethereum's ecrecover and Bitcoin
// compact message signatures. hash is the message digest.
func SignRecoverable(priv *PrivateKey, hash []byte) (r, s *big.Int, recoveryID int) {
	return signWithRecovery(priv, hash)
}

// RecoverPubKey reconstructs the public key that produced signature (r, s) over
// hash, using recoveryID (in [0,3]) as returned by SignRecoverable. It returns
// an error if the inputs are out of range or no valid point can be recovered.
//
// Callers that do not trust the recovery id must still verify the returned key
// against the signature (RecoverPubKey guarantees the key is on the curve and
// consistent with r, but any of several keys could be recovered from crafted
// inputs); the recovered key already satisfies Verify by construction.
func RecoverPubKey(hash []byte, r, s *big.Int, recoveryID int) (*PublicKey, error) {
	if recoveryID < 0 || recoveryID > 3 {
		return nil, errBadKey("recovery id %d out of range [0,3]", recoveryID)
	}

	if r == nil || s == nil || r.Sign() <= 0 || r.Cmp(orderN) >= 0 || s.Sign() <= 0 || s.Cmp(orderN) >= 0 {
		return nil, errBadKey("signature values out of range [1, N-1]")
	}

	// Reconstruct the nonce point R. Its x is r, optionally plus N when the
	// original x overflowed the group order (recoveryID bit 1).
	x := new(big.Int).Set(r)
	if recoveryID&2 != 0 {
		x.Add(x, orderN)
		if x.Cmp(primeP) >= 0 {
			return nil, errBadKey("recovered x coordinate out of field range")
		}
	}

	y, err := decompressY(x, recoveryID&1 == 1)
	if err != nil {
		return nil, err
	}

	// Q = r^-1 * (s*R - e*G)
	rInv := new(big.Int).ModInverse(r, orderN)
	if rInv == nil {
		return nil, errBadKey("r is not invertible")
	}

	e := hashToInt(hash)
	rPoint := affineToJacobian(x, y)

	// s*R
	sR := scalarMult(rPoint, s)

	// -e*G = (N-e)*G, reduced mod N
	negE := new(big.Int).Sub(orderN, new(big.Int).Mod(e, orderN))
	negE.Mod(negE, orderN)
	negEG := scalarMult(affineToJacobian(gX, gY), negE)

	sum := sR.add(negEG)
	qx, qy := scalarMult(sum, rInv).toAffine()

	if !isOnCurve(qx, qy) {
		return nil, errBadKey("recovered point is not on the curve")
	}

	return &PublicKey{X: qx, Y: qy}, nil
}
