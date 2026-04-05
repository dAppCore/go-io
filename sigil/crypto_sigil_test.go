package sigil

import (
	"bytes"
	"crypto/rand"
	goio "io"
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCryptoSigil_XORObfuscator_RoundTrip_Good(t *testing.T) {
	ob := &XORObfuscator{}
	data := []byte("the axioms are in the weights")
	entropy := []byte("deterministic-nonce-24bytes!")

	obfuscated := ob.Obfuscate(data, entropy)
	assert.NotEqual(t, data, obfuscated)
	assert.Len(t, obfuscated, len(data))

	restored := ob.Deobfuscate(obfuscated, entropy)
	assert.Equal(t, data, restored)
}

func TestCryptoSigil_XORObfuscator_DifferentEntropyDifferentOutput_Good(t *testing.T) {
	ob := &XORObfuscator{}
	data := []byte("same plaintext")

	out1 := ob.Obfuscate(data, []byte("entropy-a"))
	out2 := ob.Obfuscate(data, []byte("entropy-b"))
	assert.NotEqual(t, out1, out2)
}

func TestCryptoSigil_XORObfuscator_Deterministic_Good(t *testing.T) {
	ob := &XORObfuscator{}
	data := []byte("reproducible")
	entropy := []byte("fixed-seed")

	out1 := ob.Obfuscate(data, entropy)
	out2 := ob.Obfuscate(data, entropy)
	assert.Equal(t, out1, out2)
}

func TestCryptoSigil_XORObfuscator_LargeData_Good(t *testing.T) {
	ob := &XORObfuscator{}
	data := make([]byte, 256)
	for i := range data {
		data[i] = byte(i)
	}
	entropy := []byte("test-entropy")

	obfuscated := ob.Obfuscate(data, entropy)
	restored := ob.Deobfuscate(obfuscated, entropy)
	assert.Equal(t, data, restored)
}

func TestCryptoSigil_XORObfuscator_EmptyData_Good(t *testing.T) {
	ob := &XORObfuscator{}
	result := ob.Obfuscate([]byte{}, []byte("entropy"))
	assert.Equal(t, []byte{}, result)

	result = ob.Deobfuscate([]byte{}, []byte("entropy"))
	assert.Equal(t, []byte{}, result)
}

func TestCryptoSigil_XORObfuscator_SymmetricProperty_Good(t *testing.T) {
	ob := &XORObfuscator{}
	data := []byte("XOR is its own inverse")
	entropy := []byte("nonce")

	double := ob.Obfuscate(ob.Obfuscate(data, entropy), entropy)
	assert.Equal(t, data, double)
}

func TestCryptoSigil_ShuffleMaskObfuscator_RoundTrip_Good(t *testing.T) {
	ob := &ShuffleMaskObfuscator{}
	data := []byte("shuffle and mask protect patterns")
	entropy := []byte("deterministic-entropy")

	obfuscated := ob.Obfuscate(data, entropy)
	assert.NotEqual(t, data, obfuscated)
	assert.Len(t, obfuscated, len(data))

	restored := ob.Deobfuscate(obfuscated, entropy)
	assert.Equal(t, data, restored)
}

func TestCryptoSigil_ShuffleMaskObfuscator_DifferentEntropy_Good(t *testing.T) {
	ob := &ShuffleMaskObfuscator{}
	data := []byte("same data")

	out1 := ob.Obfuscate(data, []byte("entropy-1"))
	out2 := ob.Obfuscate(data, []byte("entropy-2"))
	assert.NotEqual(t, out1, out2)
}

func TestCryptoSigil_ShuffleMaskObfuscator_Deterministic_Good(t *testing.T) {
	ob := &ShuffleMaskObfuscator{}
	data := []byte("reproducible shuffle")
	entropy := []byte("fixed")

	out1 := ob.Obfuscate(data, entropy)
	out2 := ob.Obfuscate(data, entropy)
	assert.Equal(t, out1, out2)
}

func TestCryptoSigil_ShuffleMaskObfuscator_LargeData_Good(t *testing.T) {
	ob := &ShuffleMaskObfuscator{}
	data := make([]byte, 512)
	for i := range data {
		data[i] = byte(i % 256)
	}
	entropy := []byte("large-data-test")

	obfuscated := ob.Obfuscate(data, entropy)
	restored := ob.Deobfuscate(obfuscated, entropy)
	assert.Equal(t, data, restored)
}

func TestCryptoSigil_ShuffleMaskObfuscator_EmptyData_Good(t *testing.T) {
	ob := &ShuffleMaskObfuscator{}
	result := ob.Obfuscate([]byte{}, []byte("entropy"))
	assert.Equal(t, []byte{}, result)

	result = ob.Deobfuscate([]byte{}, []byte("entropy"))
	assert.Equal(t, []byte{}, result)
}

func TestCryptoSigil_ShuffleMaskObfuscator_SingleByte_Good(t *testing.T) {
	ob := &ShuffleMaskObfuscator{}
	data := []byte{0x42}
	entropy := []byte("single")

	obfuscated := ob.Obfuscate(data, entropy)
	restored := ob.Deobfuscate(obfuscated, entropy)
	assert.Equal(t, data, restored)
}

func TestCryptoSigil_NewChaChaPolySigil_Good(t *testing.T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	cipherSigil, err := NewChaChaPolySigil(key, nil)
	require.NoError(t, err)
	assert.NotNil(t, cipherSigil)
	assert.Equal(t, key, cipherSigil.Key())
	assert.NotNil(t, cipherSigil.Obfuscator())
}

func TestCryptoSigil_NewChaChaPolySigil_KeyIsCopied_Good(t *testing.T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)
	original := make([]byte, 32)
	copy(original, key)

	cipherSigil, err := NewChaChaPolySigil(key, nil)
	require.NoError(t, err)

	key[0] ^= 0xFF
	assert.Equal(t, original, cipherSigil.Key())
}

func TestCryptoSigil_NewChaChaPolySigil_ShortKey_Bad(t *testing.T) {
	_, err := NewChaChaPolySigil([]byte("too short"), nil)
	assert.ErrorIs(t, err, InvalidKeyError)
}

func TestCryptoSigil_NewChaChaPolySigil_LongKey_Bad(t *testing.T) {
	_, err := NewChaChaPolySigil(make([]byte, 64), nil)
	assert.ErrorIs(t, err, InvalidKeyError)
}

func TestCryptoSigil_NewChaChaPolySigil_EmptyKey_Bad(t *testing.T) {
	_, err := NewChaChaPolySigil(nil, nil)
	assert.ErrorIs(t, err, InvalidKeyError)
}

func TestCryptoSigil_NewChaChaPolySigil_CustomObfuscator_Good(t *testing.T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	ob := &ShuffleMaskObfuscator{}
	cipherSigil, err := NewChaChaPolySigil(key, ob)
	require.NoError(t, err)
	assert.Equal(t, ob, cipherSigil.Obfuscator())
}

func TestCryptoSigil_NewChaChaPolySigil_CustomObfuscatorNil_Good(t *testing.T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	cipherSigil, err := NewChaChaPolySigil(key, nil)
	require.NoError(t, err)
	assert.IsType(t, &XORObfuscator{}, cipherSigil.Obfuscator())
}

func TestCryptoSigil_NewChaChaPolySigil_CustomObfuscator_InvalidKey_Bad(t *testing.T) {
	_, err := NewChaChaPolySigil([]byte("bad"), &XORObfuscator{})
	assert.ErrorIs(t, err, InvalidKeyError)
}

func TestCryptoSigil_ChaChaPolySigil_RoundTrip_Good(t *testing.T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	cipherSigil, err := NewChaChaPolySigil(key, nil)
	require.NoError(t, err)

	plaintext := []byte("consciousness does not merely avoid causing harm")
	ciphertext, err := cipherSigil.In(plaintext)
	require.NoError(t, err)
	assert.NotEqual(t, plaintext, ciphertext)
	assert.Greater(t, len(ciphertext), len(plaintext))

	decrypted, err := cipherSigil.Out(ciphertext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestCryptoSigil_ChaChaPolySigil_CustomShuffleMask_Good(t *testing.T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	cipherSigil, err := NewChaChaPolySigil(key, &ShuffleMaskObfuscator{})
	require.NoError(t, err)

	plaintext := []byte("shuffle mask pre-obfuscation layer")
	ciphertext, err := cipherSigil.In(plaintext)
	require.NoError(t, err)

	decrypted, err := cipherSigil.Out(ciphertext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestCryptoSigil_ChaChaPolySigil_NilData_Good(t *testing.T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	cipherSigil, err := NewChaChaPolySigil(key, nil)
	require.NoError(t, err)

	enc, err := cipherSigil.In(nil)
	require.NoError(t, err)
	assert.Nil(t, enc)

	dec, err := cipherSigil.Out(nil)
	require.NoError(t, err)
	assert.Nil(t, dec)
}

func TestCryptoSigil_ChaChaPolySigil_EmptyPlaintext_Good(t *testing.T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	cipherSigil, err := NewChaChaPolySigil(key, nil)
	require.NoError(t, err)

	ciphertext, err := cipherSigil.In([]byte{})
	require.NoError(t, err)
	assert.NotEmpty(t, ciphertext)

	decrypted, err := cipherSigil.Out(ciphertext)
	require.NoError(t, err)
	assert.Equal(t, []byte{}, decrypted)
}

func TestCryptoSigil_ChaChaPolySigil_DifferentCiphertextsPerCall_Good(t *testing.T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	cipherSigil, err := NewChaChaPolySigil(key, nil)
	require.NoError(t, err)

	plaintext := []byte("same input")
	ct1, _ := cipherSigil.In(plaintext)
	ct2, _ := cipherSigil.In(plaintext)

	assert.NotEqual(t, ct1, ct2)
}

func TestCryptoSigil_ChaChaPolySigil_NoKey_Bad(t *testing.T) {
	cipherSigil := &ChaChaPolySigil{}

	_, err := cipherSigil.In([]byte("data"))
	assert.ErrorIs(t, err, NoKeyConfiguredError)

	_, err = cipherSigil.Out([]byte("data"))
	assert.ErrorIs(t, err, NoKeyConfiguredError)
}

func TestCryptoSigil_ChaChaPolySigil_WrongKey_Bad(t *testing.T) {
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	_, _ = rand.Read(key1)
	_, _ = rand.Read(key2)

	cipherSigilOne, _ := NewChaChaPolySigil(key1, nil)
	cipherSigilTwo, _ := NewChaChaPolySigil(key2, nil)

	ciphertext, err := cipherSigilOne.In([]byte("secret"))
	require.NoError(t, err)

	_, err = cipherSigilTwo.Out(ciphertext)
	assert.ErrorIs(t, err, DecryptionFailedError)
}

func TestCryptoSigil_ChaChaPolySigil_TruncatedCiphertext_Bad(t *testing.T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	cipherSigil, _ := NewChaChaPolySigil(key, nil)
	_, err := cipherSigil.Out([]byte("too short"))
	assert.ErrorIs(t, err, CiphertextTooShortError)
}

func TestCryptoSigil_ChaChaPolySigil_TamperedCiphertext_Bad(t *testing.T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	cipherSigil, _ := NewChaChaPolySigil(key, nil)
	ciphertext, _ := cipherSigil.In([]byte("authentic data"))

	ciphertext[30] ^= 0xFF

	_, err := cipherSigil.Out(ciphertext)
	assert.ErrorIs(t, err, DecryptionFailedError)
}

type failReader struct{}

func (reader *failReader) Read([]byte) (int, error) {
	return 0, core.NewError("entropy source failed")
}

func TestCryptoSigil_ChaChaPolySigil_RandomReaderFailure_Bad(t *testing.T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	cipherSigil, _ := NewChaChaPolySigil(key, nil)
	cipherSigil.randomReader = &failReader{}

	_, err := cipherSigil.In([]byte("data"))
	assert.Error(t, err)
}

func TestCryptoSigil_ChaChaPolySigil_NoObfuscator_Good(t *testing.T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	cipherSigil, _ := NewChaChaPolySigil(key, nil)
	cipherSigil.SetObfuscator(nil)

	plaintext := []byte("raw encryption without pre-obfuscation")
	ciphertext, err := cipherSigil.In(plaintext)
	require.NoError(t, err)

	decrypted, err := cipherSigil.Out(ciphertext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestCryptoSigil_NonceFromCiphertext_Good(t *testing.T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	cipherSigil, _ := NewChaChaPolySigil(key, nil)
	ciphertext, _ := cipherSigil.In([]byte("nonce extraction test"))

	nonce, err := NonceFromCiphertext(ciphertext)
	require.NoError(t, err)
	assert.Len(t, nonce, 24)

	assert.Equal(t, ciphertext[:24], nonce)
}

func TestCryptoSigil_NonceFromCiphertext_NonceCopied_Good(t *testing.T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	cipherSigil, _ := NewChaChaPolySigil(key, nil)
	ciphertext, _ := cipherSigil.In([]byte("data"))

	nonce, _ := NonceFromCiphertext(ciphertext)
	original := make([]byte, len(nonce))
	copy(original, nonce)

	nonce[0] ^= 0xFF
	assert.Equal(t, original, ciphertext[:24])
}

func TestCryptoSigil_NonceFromCiphertext_TooShort_Bad(t *testing.T) {
	_, err := NonceFromCiphertext([]byte("short"))
	assert.ErrorIs(t, err, CiphertextTooShortError)
}

func TestCryptoSigil_NonceFromCiphertext_Empty_Bad(t *testing.T) {
	_, err := NonceFromCiphertext(nil)
	assert.ErrorIs(t, err, CiphertextTooShortError)
}

func TestCryptoSigil_ChaChaPolySigil_InTransmutePipeline_Good(t *testing.T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	cipherSigil, _ := NewChaChaPolySigil(key, nil)
	hexSigil, _ := NewSigil("hex")

	chain := []Sigil{cipherSigil, hexSigil}
	plaintext := []byte("encrypt then hex encode")

	encoded, err := Transmute(plaintext, chain)
	require.NoError(t, err)

	assert.True(t, isHex(encoded))

	decoded, err := Untransmute(encoded, chain)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decoded)
}

func isHex(data []byte) bool {
	for _, b := range data {
		if !((b >= '0' && b <= '9') || (b >= 'a' && b <= 'f')) {
			return false
		}
	}
	return len(data) > 0
}

type failSigil struct{}

func (sigil *failSigil) In([]byte) ([]byte, error)  { return nil, core.NewError("fail in") }
func (sigil *failSigil) Out([]byte) ([]byte, error) { return nil, core.NewError("fail out") }

func TestCryptoSigil_Transmute_ErrorPropagation_Bad(t *testing.T) {
	_, err := Transmute([]byte("data"), []Sigil{&failSigil{}})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "fail in")
}

func TestCryptoSigil_Untransmute_ErrorPropagation_Bad(t *testing.T) {
	_, err := Untransmute([]byte("data"), []Sigil{&failSigil{}})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "fail out")
}

func TestCryptoSigil_GzipSigil_CustomOutputWriter_Good(t *testing.T) {
	var outputBuffer bytes.Buffer
	gzipSigil := &GzipSigil{outputWriter: &outputBuffer}

	_, err := gzipSigil.In([]byte("test data"))
	require.NoError(t, err)
	assert.Greater(t, outputBuffer.Len(), 0)
}

func TestCryptoSigil_DeriveKeyStream_ExactBlockSize_Good(t *testing.T) {
	ob := &XORObfuscator{}
	data := make([]byte, 32)
	for i := range data {
		data[i] = byte(i)
	}
	entropy := []byte("block-boundary")

	obfuscated := ob.Obfuscate(data, entropy)
	restored := ob.Deobfuscate(obfuscated, entropy)
	assert.Equal(t, data, restored)
}

func TestCryptoSigil_ChaChaPolySigil_NilRandomReader_Good(t *testing.T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	cipherSigil, _ := NewChaChaPolySigil(key, nil)
	cipherSigil.randomReader = nil

	ciphertext, err := cipherSigil.In([]byte("fallback reader"))
	require.NoError(t, err)

	decrypted, err := cipherSigil.Out(ciphertext)
	require.NoError(t, err)
	assert.Equal(t, []byte("fallback reader"), decrypted)
}

type limitReader struct {
	data []byte
	pos  int
}

func (l *limitReader) Read(p []byte) (int, error) {
	if l.pos >= len(l.data) {
		return 0, goio.EOF
	}
	bytesCopied := copy(p, l.data[l.pos:])
	l.pos += bytesCopied
	return bytesCopied, nil
}
