package nanoproto

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"golang.org/x/crypto/blake2b"
)

var nanoAlphabet = "13456789abcdefghijkmnopqrstuwxyz"

var nanoAlphabetMap = func() map[rune]int {
	m := make(map[rune]int)
	for i, c := range nanoAlphabet {
		m[c] = i
	}
	return m
}()

// Credits: ChatGPT
func convertBalanceToBytes(balanceStr string) ([]byte, error) {
	var balanceBytes []byte = make([]byte, 16)

	// Convert string to big.Int
	balanceBigInt := new(big.Int)
	balanceBigInt, ok := balanceBigInt.SetString(balanceStr, 10) // Base 10 conversion
	if !ok {
		return balanceBytes, fmt.Errorf("failed to convert balance string to big.Int")
	}

	// Convert big.Int to 16-byte big-endian representation
	balanceBin := balanceBigInt.Bytes()                 // Get byte slice representation
	copy(balanceBytes[16-len(balanceBin):], balanceBin) // Right-align in 16 bytes

	return balanceBytes, nil
}

// Credits: ChatGPT
func nanoAddressToPublicKey(addr string) (string, error) {
	var prefix string
	if strings.HasPrefix(addr, "nano_") {
		prefix = "nano_"
	} else if strings.HasPrefix(addr, "xrb_") {
		prefix = "xrb_"
	} else {
		return "", errors.New("invalid address prefix")
	}

	// Remove the prefix.
	addrBody := addr[len(prefix):]
	// Nano addresses should have exactly 60 characters after the prefix:
	// 52 for the public key and 8 for the checksum.
	if len(addrBody) != 60 {
		return "", errors.New("invalid address length")
	}

	// Split into the encoded public key and the provided checksum.
	encodedPubKey := addrBody[:52]
	encodedChecksum := addrBody[52:]

	// Decode the 52-character encoded public key into a big.Int.
	pubKeyInt := big.NewInt(0)
	for _, c := range encodedPubKey {
		val, ok := nanoAlphabetMap[c]
		if !ok {
			return "", fmt.Errorf("invalid character in address: %c", c)
		}
		pubKeyInt.Mul(pubKeyInt, big.NewInt(32))
		pubKeyInt.Add(pubKeyInt, big.NewInt(int64(val)))
	}

	// The decoded number is 260 bits; the first 4 bits must be zero.
	// We extract the lower 256 bits as the actual public key.
	mod := new(big.Int).Lsh(big.NewInt(1), 256) // 1 << 256
	pubKeyInt.Mod(pubKeyInt, mod)

	// Convert the public key into a 32-byte slice (pad with zeros if needed).
	pubKeyBytes := pubKeyInt.Bytes()
	if len(pubKeyBytes) < 32 {
		padded := make([]byte, 32)
		copy(padded[32-len(pubKeyBytes):], pubKeyBytes)
		pubKeyBytes = padded
	}

	// Recompute the checksum:
	// 1. Compute a 5-byte Blake2b hash of the public key.
	checksumHasher, err := blake2b.New(5, nil)
	if err != nil {
		return "", err
	}
	checksumHasher.Write(pubKeyBytes)
	checksum := checksumHasher.Sum(nil)

	// 2. Reverse the checksum bytes.
	for i, j := 0, len(checksum)-1; i < j; i, j = i+1, j-1 {
		checksum[i], checksum[j] = checksum[j], checksum[i]
	}

	// 3. Convert the 5-byte checksum into 8 characters using Nano’s base32 alphabet.
	csInt := new(big.Int).SetBytes(checksum)
	checksumDigits := make([]byte, 8)
	base := big.NewInt(32)
	// Since 5 bytes = 40 bits, we extract 8 groups of 5 bits.
	for i := 7; i >= 0; i-- {
		var rem big.Int
		csInt.DivMod(csInt, base, &rem)
		checksumDigits[i] = nanoAlphabet[rem.Int64()]
	}
	encodedCalcChecksum := string(checksumDigits)

	// Validate that the calculated checksum matches the address’s checksum.
	if encodedCalcChecksum != encodedChecksum {
		return "", errors.New("invalid checksum")
	}

	return hex.EncodeToString(pubKeyBytes), nil
}

// Credits: ChatGPT
func publicKeyToNanoAddress(pubKey []byte) (string, error) {
	// Verify public key length.
	if len(pubKey) != 32 {
		return "", errors.New("public key must be 32 bytes")
	}

	// Convert public key to a big integer.
	keyInt := new(big.Int).SetBytes(pubKey)

	// Encode the public key into a 52-character string using Nano's custom base32.
	// This represents a 260-bit number (the public key with 4 leading zero bits).
	encodedKey := make([]byte, 52)
	base := big.NewInt(32)
	temp := new(big.Int).Set(keyInt)
	for i := 51; i >= 0; i-- {
		mod := new(big.Int)
		temp.QuoRem(temp, base, mod)
		encodedKey[i] = nanoAlphabet[mod.Int64()]
	}

	// Compute the checksum: a 5-byte Blake2b hash of the public key.
	checksumHasher, err := blake2b.New(5, nil)
	if err != nil {
		return "", err
	}
	checksumHasher.Write(pubKey)
	checksum := checksumHasher.Sum(nil)

	// Reverse the checksum bytes.
	for i, j := 0, len(checksum)-1; i < j; i, j = i+1, j-1 {
		checksum[i], checksum[j] = checksum[j], checksum[i]
	}

	// Encode the checksum into an 8-character string using Nano's custom base32.
	csInt := new(big.Int).SetBytes(checksum)
	encodedChecksum := make([]byte, 8)
	for i := 7; i >= 0; i-- {
		mod := new(big.Int)
		csInt.QuoRem(csInt, base, mod)
		encodedChecksum[i] = nanoAlphabet[mod.Int64()]
	}

	// Construct the final Nano address.
	address := "nano_" + string(encodedKey) + string(encodedChecksum)
	return address, nil
}

// These functions appeared here in 2023, I'm not sure if they are used in the codebase.

func prependLength(buf *[]byte) []byte {
	buffer := bytes.NewBuffer([]byte{})
	hexStr := fmt.Sprintf("%02X", len(*buf))
	out, err := hex.DecodeString(strings.Repeat("0", 10-len(hexStr)) + hexStr)

	if err != nil {
		panic(err)
	}

	buffer.Write(out)
	buffer.Write(*buf)

	return buffer.Bytes()
}

func chunks(buf *[]byte, n int) [][]byte {
	var chunks [][]byte
	length := len(*buf)

	for i := 0; i < length; i += n {
		to := i + n

		if to > length {
			to = length
		}

		chunks = append(chunks, (*buf)[i:to])
	}

	return chunks
}
