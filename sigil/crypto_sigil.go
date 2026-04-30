// Example: cipherSigil, _ := sigil.NewChaChaPolySigil([]byte("0123456789abcdef0123456789abcdef"), nil)
// Example: ciphertext, _ := cipherSigil.In([]byte("payload"))
// Example: plaintext, _ := cipherSigil.Out(ciphertext)
package sigil

import (
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	goio "io"

	core "dappco.re/go"
	"golang.org/x/crypto/chacha20poly1305"
)

var (
	// Example: errors.Is(err, sigil.InvalidKeyError)
	InvalidKeyError = core.E("sigil.InvalidKeyError", "invalid key size, must be 32 bytes", nil)

	// Example: errors.Is(err, sigil.InvalidNonceError)
	InvalidNonceError = core.E("sigil.InvalidNonceError", "invalid nonce argument; use PreObfuscator or nil", nil)

	// Example: errors.Is(err, sigil.CiphertextTooShortError)
	CiphertextTooShortError = core.E("sigil.CiphertextTooShortError", "ciphertext too short", nil)

	// Example: errors.Is(err, sigil.DecryptionFailedError)
	DecryptionFailedError = core.E("sigil.DecryptionFailedError", "decryption failed", nil)

	// Example: errors.Is(err, sigil.NoKeyConfiguredError)
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
	return obfuscator.Obfuscate(data, entropy)
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
	hashFunction := sha256.New()

	blockNum := uint64(0)
	offset := 0
	for offset < length {
		hashFunction.Reset()
		hashFunction.Write(entropy)
		var blockBytes [8]byte
		binary.BigEndian.PutUint64(blockBytes[:], blockNum)
		hashFunction.Write(blockBytes[:])
		block := hashFunction.Sum(nil)

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

	permutation := obfuscator.generatePermutation(entropy, len(data))
	mask := obfuscator.deriveMask(entropy, len(data))

	for i := range result {
		result[i] ^= mask[i]
	}

	shuffled := make([]byte, len(data))
	for destinationIndex, sourceIndex := range permutation {
		shuffled[destinationIndex] = result[sourceIndex]
	}

	return shuffled
}

func (obfuscator *ShuffleMaskObfuscator) Deobfuscate(data []byte, entropy []byte) []byte {
	if len(data) == 0 {
		return data
	}

	result := make([]byte, len(data))

	permutation := obfuscator.generatePermutation(entropy, len(data))
	mask := obfuscator.deriveMask(entropy, len(data))

	for destinationIndex, sourceIndex := range permutation {
		result[sourceIndex] = data[destinationIndex]
	}

	for i := range result {
		result[i] ^= mask[i]
	}

	return result
}

func (obfuscator *ShuffleMaskObfuscator) generatePermutation(entropy []byte, length int) []int {
	permutation := make([]int, length)
	for i := range permutation {
		permutation[i] = i
	}

	hashFunction := sha256.New()
	hashFunction.Write(entropy)
	hashFunction.Write([]byte("permutation"))
	seed := hashFunction.Sum(nil)

	for i := length - 1; i > 0; i-- {
		hashFunction.Reset()
		hashFunction.Write(seed)
		var iBytes [8]byte
		binary.BigEndian.PutUint64(iBytes[:], uint64(i))
		hashFunction.Write(iBytes[:])
		jBytes := hashFunction.Sum(nil)
		j := int(binary.BigEndian.Uint64(jBytes[:8]) % uint64(i+1))
		permutation[i], permutation[j] = permutation[j], permutation[i]
	}

	return permutation
}

func (obfuscator *ShuffleMaskObfuscator) deriveMask(entropy []byte, length int) []byte {
	mask := make([]byte, length)
	hashFunction := sha256.New()

	blockNum := uint64(0)
	offset := 0
	for offset < length {
		hashFunction.Reset()
		hashFunction.Write(entropy)
		hashFunction.Write([]byte("mask"))
		var blockBytes [8]byte
		binary.BigEndian.PutUint64(blockBytes[:], blockNum)
		hashFunction.Write(blockBytes[:])
		block := hashFunction.Sum(nil)

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
	key          []byte
	nonceSize    int
	obfuscator   PreObfuscator
	randomReader goio.Reader
}

// Example: key := cipherSigil.Key()
func (s *ChaChaPolySigil) Key() []byte {
	result := make([]byte, len(s.key))
	copy(result, s.key)
	return result
}

// Nonce returns nil. Encryption nonces are generated per message by In and
// prepended to the ciphertext.
func (s *ChaChaPolySigil) Nonce() []byte {
	return nil
}

// Example: ob := cipherSigil.Obfuscator()
func (s *ChaChaPolySigil) Obfuscator() PreObfuscator {
	return s.obfuscator
}

// Example: cipherSigil.SetObfuscator(nil)
func (s *ChaChaPolySigil) SetObfuscator(obfuscator PreObfuscator) {
	s.obfuscator = obfuscator
}

// NewChaChaPolySigil creates a ChaCha20-Poly1305 sigil. The nonce argument is
// retained for API compatibility; pass nil for the default pre-obfuscator or a
// PreObfuscator for custom pre-obfuscation. Fixed []byte nonces are rejected:
// ChaCha20-Poly1305 catastrophically fails under nonce reuse, leaking plaintext
// relationships and enabling authenticated forgeries. In always generates a
// fresh random nonce and prepends it to the ciphertext.
//
// WARNING: when using a custom PreObfuscator, nonce uniqueness remains the
// caller's responsibility. The PreObfuscator must treat the supplied entropy as
// a per-message nonce and must not introduce deterministic nonce reuse.
func NewChaChaPolySigil(key []byte, nonce any) (*ChaChaPolySigil, error) {
	if len(key) != 32 {
		return nil, InvalidKeyError
	}

	keyCopy := make([]byte, 32)
	copy(keyCopy, key)

	sigil := &ChaChaPolySigil{
		key:          keyCopy,
		nonceSize:    chacha20poly1305.NonceSizeX,
		randomReader: rand.Reader,
	}

	switch value := nonce.(type) {
	case nil:
		sigil.obfuscator = &XORObfuscator{}
	case []byte:
		return nil, core.E("sigil.NewChaChaPolySigil", "fixed-nonce []byte path removed; use PreObfuscator or nil", InvalidNonceError)
	case PreObfuscator:
		if value == nil {
			sigil.obfuscator = &XORObfuscator{}
			return sigil, nil
		}
		sigil.obfuscator = value
	default:
		return nil, core.E("sigil.NewChaChaPolySigil", "nonce must be PreObfuscator or nil", InvalidNonceError)
	}

	return sigil, nil
}

func (s *ChaChaPolySigil) In(data []byte) ([]byte, error) {
	if s.key == nil {
		return nil, NoKeyConfiguredError
	}
	if data == nil {
		return nil, nil
	}

	aead, err := s.newAEAD()
	if err != nil {
		return nil, core.E("sigil.ChaChaPolySigil.In", "create cipher", err)
	}

	nonce := make([]byte, aead.NonceSize())
	reader := s.randomReader
	if reader == nil {
		reader = rand.Reader
	}
	if _, err := goio.ReadFull(reader, nonce); err != nil {
		return nil, core.E("sigil.ChaChaPolySigil.In", "read nonce", err)
	}

	obfuscated := data
	if s.obfuscator != nil {
		obfuscated = s.obfuscator.Obfuscate(data, cloneBytes(nonce))
	}

	ciphertext := aead.Seal(nonce, nonce, obfuscated, nil)

	return ciphertext, nil
}

func (s *ChaChaPolySigil) Out(data []byte) ([]byte, error) {
	if s.key == nil {
		return nil, NoKeyConfiguredError
	}
	if data == nil {
		return nil, nil
	}

	aead, err := s.newAEAD()
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
		// The underlying aead error is intentionally hidden: surfacing raw AEAD errors can
		// leak oracle information to an attacker. DecryptionFailedError is the safe sentinel.
		return nil, core.E("sigil.ChaChaPolySigil.Out", "decrypt ciphertext", DecryptionFailedError)
	}

	plaintext := obfuscated
	if s.obfuscator != nil {
		plaintext = s.obfuscator.Deobfuscate(obfuscated, cloneBytes(nonce))
	}

	if len(plaintext) == 0 {
		return []byte{}, nil
	}

	return plaintext, nil
}

func (s *ChaChaPolySigil) newAEAD() (cipher.AEAD, error) {
	switch s.activeNonceSize() {
	case chacha20poly1305.NonceSize:
		return chacha20poly1305.New(s.key)
	case chacha20poly1305.NonceSizeX:
		return chacha20poly1305.NewX(s.key)
	default:
		return nil, InvalidNonceError
	}
}

func (s *ChaChaPolySigil) activeNonceSize() int {
	if s.nonceSize != 0 {
		return s.nonceSize
	}
	return chacha20poly1305.NonceSizeX
}

func cloneBytes(data []byte) []byte {
	result := make([]byte, len(data))
	copy(result, data)
	return result
}

// Example: nonce, _ := sigil.NonceFromCiphertext(ciphertext)
func NonceFromCiphertext(ciphertext []byte) ([]byte, error) {
	nonceSize := chacha20poly1305.NonceSizeX
	if len(ciphertext) < nonceSize {
		return nil, CiphertextTooShortError
	}
	nonceCopy := make([]byte, nonceSize)
	copy(nonceCopy, ciphertext[:nonceSize])
	return nonceCopy, nil
}
