// Example: cipherSigil, _ := sigil.NewChaChaPolySigil([]byte("0123456789abcdef0123456789abcdef"))
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
	// InvalidKeyError is returned when the encryption key is not 32 bytes.
	InvalidKeyError = core.E("sigil.InvalidKeyError", "invalid key size, must be 32 bytes", nil)

	// CiphertextTooShortError is returned when the ciphertext is too short to decrypt.
	CiphertextTooShortError = core.E("sigil.CiphertextTooShortError", "ciphertext too short", nil)

	// DecryptionFailedError is returned when decryption or authentication fails.
	DecryptionFailedError = core.E("sigil.DecryptionFailedError", "decryption failed", nil)

	// NoKeyConfiguredError is returned when no encryption key has been set.
	NoKeyConfiguredError = core.E("sigil.NoKeyConfiguredError", "no encryption key configured", nil)
)

// PreObfuscator customises the bytes mixed in before and after encryption.
type PreObfuscator interface {
	// Obfuscate transforms plaintext before encryption using the provided entropy.
	// The entropy is typically the encryption nonce, ensuring the transformation
	// is unique per-encryption without additional random generation.
	Obfuscate(data []byte, entropy []byte) []byte

	// Deobfuscate reverses the transformation after decryption.
	// Must be called with the same entropy used during Obfuscate.
	Deobfuscate(data []byte, entropy []byte) []byte
}

// Example: cipherSigil, _ := sigil.NewChaChaPolySigil(key)
type XORObfuscator struct{}

// Obfuscate XORs the data with a key stream derived from the entropy.
func (obfuscator *XORObfuscator) Obfuscate(data []byte, entropy []byte) []byte {
	if len(data) == 0 {
		return data
	}
	return obfuscator.transform(data, entropy)
}

// Deobfuscate reverses the XOR transformation (XOR is symmetric).
func (obfuscator *XORObfuscator) Deobfuscate(data []byte, entropy []byte) []byte {
	if len(data) == 0 {
		return data
	}
	return obfuscator.transform(data, entropy)
}

// transform applies XOR with an entropy-derived key stream.
func (obfuscator *XORObfuscator) transform(data []byte, entropy []byte) []byte {
	result := make([]byte, len(data))
	keyStream := obfuscator.deriveKeyStream(entropy, len(data))
	for i := range data {
		result[i] = data[i] ^ keyStream[i]
	}
	return result
}

// deriveKeyStream creates a deterministic key stream from entropy.
func (obfuscator *XORObfuscator) deriveKeyStream(entropy []byte, length int) []byte {
	stream := make([]byte, length)
	h := sha256.New()

	// Generate key stream in 32-byte blocks
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

// ShuffleMaskObfuscator adds byte shuffling on top of XOR masking.
type ShuffleMaskObfuscator struct{}

// Obfuscate shuffles bytes and applies a mask derived from entropy.
func (obfuscator *ShuffleMaskObfuscator) Obfuscate(data []byte, entropy []byte) []byte {
	if len(data) == 0 {
		return data
	}

	result := make([]byte, len(data))
	copy(result, data)

	// Generate permutation and mask from entropy
	perm := obfuscator.generatePermutation(entropy, len(data))
	mask := obfuscator.deriveMask(entropy, len(data))

	// Apply mask first, then shuffle
	for i := range result {
		result[i] ^= mask[i]
	}

	// Shuffle using Fisher-Yates with deterministic seed
	shuffled := make([]byte, len(data))
	for i, p := range perm {
		shuffled[i] = result[p]
	}

	return shuffled
}

// Deobfuscate reverses the shuffle and mask operations.
func (obfuscator *ShuffleMaskObfuscator) Deobfuscate(data []byte, entropy []byte) []byte {
	if len(data) == 0 {
		return data
	}

	result := make([]byte, len(data))

	// Generate permutation and mask from entropy
	perm := obfuscator.generatePermutation(entropy, len(data))
	mask := obfuscator.deriveMask(entropy, len(data))

	// Unshuffle first
	for i, p := range perm {
		result[p] = data[i]
	}

	// Remove mask
	for i := range result {
		result[i] ^= mask[i]
	}

	return result
}

// generatePermutation creates a deterministic permutation from entropy.
func (obfuscator *ShuffleMaskObfuscator) generatePermutation(entropy []byte, length int) []int {
	perm := make([]int, length)
	for i := range perm {
		perm[i] = i
	}

	// Use entropy to seed a deterministic shuffle
	h := sha256.New()
	h.Write(entropy)
	h.Write([]byte("permutation"))
	seed := h.Sum(nil)

	// Fisher-Yates shuffle with deterministic randomness
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

// deriveMask creates a mask byte array from entropy.
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

// Example: cipherSigil, _ := sigil.NewChaChaPolySigil(key)
// Example: cipherSigil, _ := sigil.NewChaChaPolySigilWithObfuscator(key, &sigil.ShuffleMaskObfuscator{})
type ChaChaPolySigil struct {
	Key          []byte
	Obfuscator   PreObfuscator
	randomReader goio.Reader // for testing injection
}

// Example: cipherSigil, _ := sigil.NewChaChaPolySigil([]byte("0123456789abcdef0123456789abcdef"))
// ciphertext, _ := cipherSigil.In([]byte("payload"))
// plaintext, _ := cipherSigil.Out(ciphertext)
func NewChaChaPolySigil(key []byte) (*ChaChaPolySigil, error) {
	if len(key) != 32 {
		return nil, InvalidKeyError
	}

	keyCopy := make([]byte, 32)
	copy(keyCopy, key)

	return &ChaChaPolySigil{
		Key:          keyCopy,
		Obfuscator:   &XORObfuscator{},
		randomReader: rand.Reader,
	}, nil
}

// Example: cipherSigil, _ := sigil.NewChaChaPolySigilWithObfuscator(
//
//	[]byte("0123456789abcdef0123456789abcdef"),
//	&sigil.ShuffleMaskObfuscator{},
//
// )
// ciphertext, _ := cipherSigil.In([]byte("payload"))
// plaintext, _ := cipherSigil.Out(ciphertext)
func NewChaChaPolySigilWithObfuscator(key []byte, obfuscator PreObfuscator) (*ChaChaPolySigil, error) {
	cipherSigil, err := NewChaChaPolySigil(key)
	if err != nil {
		return nil, err
	}
	if obfuscator != nil {
		cipherSigil.Obfuscator = obfuscator
	}
	return cipherSigil, nil
}

// In encrypts plaintext with the configured pre-obfuscator.
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

	// Generate nonce
	nonce := make([]byte, aead.NonceSize())
	reader := sigil.randomReader
	if reader == nil {
		reader = rand.Reader
	}
	if _, err := goio.ReadFull(reader, nonce); err != nil {
		return nil, core.E("sigil.ChaChaPolySigil.In", "read nonce", err)
	}

	// Pre-obfuscate the plaintext using nonce as entropy
	// This ensures CPU encryption routines never see raw plaintext
	obfuscated := data
	if sigil.Obfuscator != nil {
		obfuscated = sigil.Obfuscator.Obfuscate(data, nonce)
	}

	// Encrypt the obfuscated data
	// Output: [nonce | ciphertext | auth tag]
	ciphertext := aead.Seal(nonce, nonce, obfuscated, nil)

	return ciphertext, nil
}

// Out decrypts ciphertext and reverses the pre-obfuscation step.
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

	// Extract nonce from ciphertext
	nonce := data[:aead.NonceSize()]
	ciphertext := data[aead.NonceSize():]

	// Decrypt
	obfuscated, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, core.E("sigil.ChaChaPolySigil.Out", "decrypt ciphertext", DecryptionFailedError)
	}

	// Deobfuscate using the same nonce as entropy
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
