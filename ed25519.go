package nanoproto

import (
	"encoding/hex"
	"errors"
	"fmt"

	"filippo.io/edwards25519"
	"golang.org/x/crypto/blake2b"
)

// Credits: DeepSeek

type Ed25519 struct{}

func NewEd25519() *Ed25519 {
	return &Ed25519{}
}

// GenerateKeys generates an Ed25519 key pair from a 32-byte seed using Blake2b for hashing.
func (e *Ed25519) GenerateKeys(seed string) (map[string]string, error) {
	seedBytes, err := hex.DecodeString(seed)
	if err != nil {
		return nil, fmt.Errorf("failed to decode seed: %v", err)
	}
	if len(seedBytes) != 32 {
		return nil, errors.New("seed must be 32 bytes")
	}

	// Hash seed with Blake2b-512 and take first 32 bytes
	hash := blake2b.Sum512(seedBytes)
	h := hash[:32]

	// Clamp the scalar
	h[0] &= 0xf8
	h[31] &= 0x7f
	h[31] |= 0x40

	// Generate public key
	scalar, err := edwards25519.NewScalar().SetBytesWithClamping(h)
	if err != nil {
		return nil, fmt.Errorf("failed to create scalar: %v", err)
	}
	publicKeyPoint := new(edwards25519.Point).ScalarBaseMult(scalar)
	publicKey := publicKeyPoint.Bytes()

	return map[string]string{
		"privateKey": seed,
		"publicKey":  hex.EncodeToString(publicKey),
	}, nil
}

// ConvertKeys converts Ed25519 keys to Curve25519 keys.
func (e *Ed25519) ConvertKeys(keyPair map[string]string) (map[string]string, error) {
	// Convert public key
	edPubKey, err := hex.DecodeString(keyPair["publicKey"])
	if err != nil {
		return nil, fmt.Errorf("failed to decode public key: %v", err)
	}
	edPoint, err := new(edwards25519.Point).SetBytes(edPubKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %v", err)
	}
	curvePub := edPoint.BytesMontgomery()

	// Convert private key (seed to scalar)
	seed, err := hex.DecodeString(keyPair["privateKey"])
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key: %v", err)
	}
	hash := blake2b.Sum512(seed)
	h := hash[:32]
	h[0] &= 0xf8
	h[31] &= 0x7f
	h[31] |= 0x40

	return map[string]string{
		"publicKey":  hex.EncodeToString(curvePub),
		"privateKey": hex.EncodeToString(h),
	}, nil
}

// Sign generates a signature for the message using the private key (seed).
func (e *Ed25519) Sign(msg []byte, privateKey []byte) ([]byte, error) {
	if len(privateKey) != 32 {
		return nil, errors.New("invalid private key length")
	}

	// Derive the secret scalar 'a' from privateKey
	hashD := blake2b.Sum512(privateKey)
	d := hashD[:32]
	d[0] &= 0xf8
	d[31] &= 0x7f
	d[31] |= 0x40
	aScalar, err := edwards25519.NewScalar().SetBytesWithClamping(d)
	if err != nil {
		return nil, fmt.Errorf("failed to create scalar: %v", err)
	}

	// Compute nonce 'r'
	rHash, err := blake2b.New512(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create hash: %v", err)
	}
	rHash.Write(d[32:64])
	rHash.Write(msg)
	rDigest := rHash.Sum(nil)
	rScalar, err := edwards25519.NewScalar().SetUniformBytes(rDigest)
	if err != nil {
		return nil, fmt.Errorf("failed to create scalar: %v", err)
	}

	// Compute R = r * Base
	R := new(edwards25519.Point).ScalarBaseMult(rScalar)

	// Compute public key A = a * Base
	A := new(edwards25519.Point).ScalarBaseMult(aScalar)

	// Compute h = Blake2b(R || A || msg)
	hHash, err := blake2b.New512(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create hash: %v", err)
	}
	hHash.Write(R.Bytes())
	hHash.Write(A.Bytes())
	hHash.Write(msg)
	hDigest := hHash.Sum(nil)
	hScalar, err := edwards25519.NewScalar().SetUniformBytes(hDigest)
	if err != nil {
		return nil, fmt.Errorf("failed to create scalar: %v", err)
	}

	// Compute S = r + h * a
	S := edwards25519.NewScalar().MultiplyAdd(hScalar, aScalar, rScalar)

	// Signature is R || S
	signature := make([]byte, 64)
	copy(signature[:32], R.Bytes())
	copy(signature[32:], S.Bytes())

	return signature, nil
}

// Verify checks the signature against the message and public key.
func (e *Ed25519) Verify(msg, publicKey, signature []byte) bool {
	if len(signature) != 64 || len(publicKey) != 32 {
		return false
	}

	R := signature[:32]
	S := signature[32:]

	// Parse public key A
	A, err := new(edwards25519.Point).SetBytes(publicKey)
	if err != nil {
		return false
	}

	// Parse S scalar
	sScalar, err := edwards25519.NewScalar().SetCanonicalBytes(S)
	if err != nil {
		return false
	}

	// Compute h = Blake2b(R || A || msg)
	hHash, err := blake2b.New512(nil)
	if err != nil {
		return false
	}
	hHash.Write(R)
	hHash.Write(publicKey)
	hHash.Write(msg)
	hDigest := hHash.Sum(nil)
	hScalar, err := edwards25519.NewScalar().SetUniformBytes(hDigest)
	if err != nil {
		return false
	}

	// Compute SB = S * Base
	SB := new(edwards25519.Point).ScalarBaseMult(sScalar)

	// Compute R + h*A
	hA := new(edwards25519.Point).ScalarMult(hScalar, A)
	RPoint, err := new(edwards25519.Point).SetBytes(R)
	if err != nil {
		return false
	}
	RhA := new(edwards25519.Point).Add(RPoint, hA)

	return SB.Equal(RhA) == 1
}
