// Example: cipherSigil, _ := sigil.NewChaChaPolySigil([]byte("0123456789abcdef0123456789abcdef"), nil)
// Example: ciphertext, _ := cipherSigil.In([]byte("payload"))
// Example: plaintext, _ := cipherSigil.Out(ciphertext)
package sigil

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	goio "io"

	core "dappco.re/go/core"
	"golang.org/x/crypto/chacha20poly1305"
)

var (
	InvalidKeyError = core.E("sigil.InvalidKeyError", "invalid key size, must be 32 bytes", nil)

	CiphertextTooShortError = core.E("sigil.CiphertextTooShortError", "ciphertext too short", nil)

	DecryptionFailedError = core.E("sigil.DecryptionFailedError", "decryption failed", nil)

	NoKeyConfiguredError = core.E("sigil.NoKeyConfiguredError", "no encryption key configured", nil)
)

// Example: obfuscator := &sigil.XORObfuscator{}
type PreObfuscator interface {
	Obfuscate(data []byte, entropy []byte) []byte

	Deobfuscate(data []byte, entropy []byte) []byte
}

// Example: obfuscator := &sigil.XORObfuscator{}
type XORObfuscator struct{}

func (obfuscator *XORObfuscator) Obfuscate(data []byte, entropy []byte) []byte {
	if len(data) == 0 {
		return data
	}
	return obfuscator.transform(data, entropy)
}

func (obfuscator *XORObfuscator) Deobfuscate(data []byte, entropy []byte) []byte {
	if len(data) == 0 {
		return data
	}
	return obfuscator.transform(data, entropy)
}

func (obfuscator *XORObfuscator) transform(data []byte, entropy []byte) []byte {
	result := make([]byte, len(data))
	keyStream := obfuscator.deriveKeyStream(entropy, len(data))
	for i := range data {
		result[i] = data[i] ^ keyStream[i]
	}
	return result
}

func (obfuscator *XORObfuscator) deriveKeyStream(entropy []byte, length int) []byte {
	stream := make([]byte, length)
	h := sha256.New()

	blockNum := uint64(0)
	offset := 0
	for offset < length {
		h.Reset()
		h.Write(entropy)
		var blockBytes [8]byte
		binary.BigEndian.PutUint64(blockBytes[:], blockNum)
		h.Write(blockBytes[:])
		block := h.Sum(nil)

		copyLen := min(len(block), length-offset)
		copy(stream[offset:], block[:copyLen])
		offset += copyLen
		blockNum++
	}
	return stream
}

// Example: obfuscator := &sigil.ShuffleMaskObfuscator{}
type ShuffleMaskObfuscator struct{}

func (obfuscator *ShuffleMaskObfuscator) Obfuscate(data []byte, entropy []byte) []byte {
	if len(data) == 0 {
		return data
	}

	result := make([]byte, len(data))
	copy(result, data)

	perm := obfuscator.generatePermutation(entropy, len(data))
	mask := obfuscator.deriveMask(entropy, len(data))

	for i := range result {
		result[i] ^= mask[i]
	}

	shuffled := make([]byte, len(data))
	for i, p := range perm {
		shuffled[i] = result[p]
	}

	return shuffled
}

func (obfuscator *ShuffleMaskObfuscator) Deobfuscate(data []byte, entropy []byte) []byte {
	if len(data) == 0 {
		return data
	}

	result := make([]byte, len(data))

	perm := obfuscator.generatePermutation(entropy, len(data))
	mask := obfuscator.deriveMask(entropy, len(data))

	for i, p := range perm {
		result[p] = data[i]
	}

	for i := range result {
		result[i] ^= mask[i]
	}

	return result
}

func (obfuscator *ShuffleMaskObfuscator) generatePermutation(entropy []byte, length int) []int {
	perm := make([]int, length)
	for i := range perm {
		perm[i] = i
	}

	h := sha256.New()
	h.Write(entropy)
	h.Write([]byte("permutation"))
	seed := h.Sum(nil)

	for i := length - 1; i > 0; i-- {
		h.Reset()
		h.Write(seed)
		var iBytes [8]byte
		binary.BigEndian.PutUint64(iBytes[:], uint64(i))
		h.Write(iBytes[:])
		jBytes := h.Sum(nil)
		j := int(binary.BigEndian.Uint64(jBytes[:8]) % uint64(i+1))
		perm[i], perm[j] = perm[j], perm[i]
	}

	return perm
}

func (obfuscator *ShuffleMaskObfuscator) deriveMask(entropy []byte, length int) []byte {
	mask := make([]byte, length)
	h := sha256.New()

	blockNum := uint64(0)
	offset := 0
	for offset < length {
		h.Reset()
		h.Write(entropy)
		h.Write([]byte("mask"))
		var blockBytes [8]byte
		binary.BigEndian.PutUint64(blockBytes[:], blockNum)
		h.Write(blockBytes[:])
		block := h.Sum(nil)

		copyLen := min(len(block), length-offset)
		copy(mask[offset:], block[:copyLen])
		offset += copyLen
		blockNum++
	}
	return mask
}

// Example: cipherSigil, _ := sigil.NewChaChaPolySigil(
// Example:     []byte("0123456789abcdef0123456789abcdef"),
// Example:     &sigil.ShuffleMaskObfuscator{},
// Example: )
type ChaChaPolySigil struct {
	Key          []byte
	Obfuscator   PreObfuscator
	randomReader goio.Reader
}

// Example: cipherSigil, _ := sigil.NewChaChaPolySigil([]byte("0123456789abcdef0123456789abcdef"), nil)
// Example: ciphertext, _ := cipherSigil.In([]byte("payload"))
// Example: plaintext, _ := cipherSigil.Out(ciphertext)
func NewChaChaPolySigil(key []byte, obfuscator PreObfuscator) (*ChaChaPolySigil, error) {
	if len(key) != 32 {
		return nil, InvalidKeyError
	}

	keyCopy := make([]byte, 32)
	copy(keyCopy, key)

	if obfuscator == nil {
		obfuscator = &XORObfuscator{}
	}

	return &ChaChaPolySigil{
		Key:          keyCopy,
		Obfuscator:   obfuscator,
		randomReader: rand.Reader,
	}, nil
}

func (sigil *ChaChaPolySigil) In(data []byte) ([]byte, error) {
	if sigil.Key == nil {
		return nil, NoKeyConfiguredError
	}
	if data == nil {
		return nil, nil
	}

	aead, err := chacha20poly1305.NewX(sigil.Key)
	if err != nil {
		return nil, core.E("sigil.ChaChaPolySigil.In", "create cipher", err)
	}

	nonce := make([]byte, aead.NonceSize())
	reader := sigil.randomReader
	if reader == nil {
		reader = rand.Reader
	}
	if _, err := goio.ReadFull(reader, nonce); err != nil {
		return nil, core.E("sigil.ChaChaPolySigil.In", "read nonce", err)
	}

	obfuscated := data
	if sigil.Obfuscator != nil {
		obfuscated = sigil.Obfuscator.Obfuscate(data, nonce)
	}

	ciphertext := aead.Seal(nonce, nonce, obfuscated, nil)

	return ciphertext, nil
}

func (sigil *ChaChaPolySigil) Out(data []byte) ([]byte, error) {
	if sigil.Key == nil {
		return nil, NoKeyConfiguredError
	}
	if data == nil {
		return nil, nil
	}

	aead, err := chacha20poly1305.NewX(sigil.Key)
	if err != nil {
		return nil, core.E("sigil.ChaChaPolySigil.Out", "create cipher", err)
	}

	minLen := aead.NonceSize() + aead.Overhead()
	if len(data) < minLen {
		return nil, CiphertextTooShortError
	}

	nonce := data[:aead.NonceSize()]
	ciphertext := data[aead.NonceSize():]

	obfuscated, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, core.E("sigil.ChaChaPolySigil.Out", "decrypt ciphertext", DecryptionFailedError)
	}

	plaintext := obfuscated
	if sigil.Obfuscator != nil {
		plaintext = sigil.Obfuscator.Deobfuscate(obfuscated, nonce)
	}

	if len(plaintext) == 0 {
		return []byte{}, nil
	}

	return plaintext, nil
}

// Example: nonce, _ := sigil.GetNonceFromCiphertext(ciphertext)
func GetNonceFromCiphertext(ciphertext []byte) ([]byte, error) {
	nonceSize := chacha20poly1305.NonceSizeX
	if len(ciphertext) < nonceSize {
		return nil, CiphertextTooShortError
	}
	nonceCopy := make([]byte, nonceSize)
	copy(nonceCopy, ciphertext[:nonceSize])
	return nonceCopy, nil
}
