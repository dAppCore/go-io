package sigil

import (
	"bytes"
	"crypto/rand"
	goio "io"

	. "dappco.re/go"
)

func TestCryptoSigil_XORObfuscator_RoundTrip_Good(t *T) {
	ob := &XORObfuscator{}
	data := []byte("the axioms are in the weights")
	entropy := []byte("deterministic-nonce-24bytes!")

	obfuscated := ob.Obfuscate(data, entropy)
	AssertNotEqual(t, data, obfuscated)
	AssertLen(t, obfuscated, len(data))

	restored := ob.Deobfuscate(obfuscated, entropy)
	AssertEqual(t, data, restored)
}

func TestCryptoSigil_XORObfuscator_DifferentEntropyDifferentOutput_Good(t *T) {
	ob := &XORObfuscator{}
	data := []byte("same plaintext")

	out1 := ob.Obfuscate(data, []byte("entropy-a"))
	out2 := ob.Obfuscate(data, []byte("entropy-b"))
	AssertNotEqual(t, out1, out2)
}

func TestCryptoSigil_XORObfuscator_Deterministic_Good(t *T) {
	ob := &XORObfuscator{}
	data := []byte("reproducible")
	entropy := []byte("fixed-seed")

	out1 := ob.Obfuscate(data, entropy)
	out2 := ob.Obfuscate(data, entropy)
	AssertEqual(t, out1, out2)
}

func TestCryptoSigil_XORObfuscator_LargeData_Good(t *T) {
	ob := &XORObfuscator{}
	data := make([]byte, 256)
	for i := range data {
		data[i] = byte(i)
	}
	entropy := []byte("test-entropy")

	obfuscated := ob.Obfuscate(data, entropy)
	restored := ob.Deobfuscate(obfuscated, entropy)
	AssertEqual(t, data, restored)
}

func TestCryptoSigil_XORObfuscator_EmptyData_Good(t *T) {
	ob := &XORObfuscator{}
	result := ob.Obfuscate([]byte{}, []byte("entropy"))
	AssertEqual(t, []byte{}, result)

	result = ob.Deobfuscate([]byte{}, []byte("entropy"))
	AssertEqual(t, []byte{}, result)
}

func TestCryptoSigil_XORObfuscator_SymmetricProperty_Good(t *T) {
	ob := &XORObfuscator{}
	data := []byte("XOR is its own inverse")
	entropy := []byte("nonce")

	double := ob.Obfuscate(ob.Obfuscate(data, entropy), entropy)
	AssertEqual(t, data, double)
}

func TestCryptoSigil_ShuffleMaskObfuscator_RoundTrip_Good(t *T) {
	ob := &ShuffleMaskObfuscator{}
	data := []byte("shuffle and mask protect patterns")
	entropy := []byte("deterministic-entropy")

	obfuscated := ob.Obfuscate(data, entropy)
	AssertNotEqual(t, data, obfuscated)
	AssertLen(t, obfuscated, len(data))

	restored := ob.Deobfuscate(obfuscated, entropy)
	AssertEqual(t, data, restored)
}

func TestCryptoSigil_ShuffleMaskObfuscator_DifferentEntropy_Good(t *T) {
	ob := &ShuffleMaskObfuscator{}
	data := []byte("same data")

	out1 := ob.Obfuscate(data, []byte("entropy-1"))
	out2 := ob.Obfuscate(data, []byte("entropy-2"))
	AssertNotEqual(t, out1, out2)
}

func TestCryptoSigil_ShuffleMaskObfuscator_Deterministic_Good(t *T) {
	ob := &ShuffleMaskObfuscator{}
	data := []byte("reproducible shuffle")
	entropy := []byte("fixed")

	out1 := ob.Obfuscate(data, entropy)
	out2 := ob.Obfuscate(data, entropy)
	AssertEqual(t, out1, out2)
}

func TestCryptoSigil_ShuffleMaskObfuscator_LargeData_Good(t *T) {
	ob := &ShuffleMaskObfuscator{}
	data := make([]byte, 512)
	for i := range data {
		data[i] = byte(i % 256)
	}
	entropy := []byte("large-data-test")

	obfuscated := ob.Obfuscate(data, entropy)
	restored := ob.Deobfuscate(obfuscated, entropy)
	AssertEqual(t, data, restored)
}

func TestCryptoSigil_ShuffleMaskObfuscator_EmptyData_Good(t *T) {
	ob := &ShuffleMaskObfuscator{}
	result := ob.Obfuscate([]byte{}, []byte("entropy"))
	AssertEqual(t, []byte{}, result)

	result = ob.Deobfuscate([]byte{}, []byte("entropy"))
	AssertEqual(t, []byte{}, result)
}

func TestCryptoSigil_ShuffleMaskObfuscator_SingleByte_Good(t *T) {
	ob := &ShuffleMaskObfuscator{}
	data := []byte{0x42}
	entropy := []byte("single")

	obfuscated := ob.Obfuscate(data, entropy)
	restored := ob.Deobfuscate(obfuscated, entropy)
	AssertEqual(t, data, restored)
}

func TestCryptoSigil_NewChaChaPolySigil_Good(t *T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	cipherSigil, err := NewChaChaPolySigil(key, nil)
	RequireNoError(t, err)
	AssertNotNil(t, cipherSigil)
	AssertEqual(t, key, cipherSigil.Key())
	AssertNotNil(t, cipherSigil.Obfuscator())
}

func TestCryptoSigil_NewChaChaPolySigil_KeyIsCopied_Good(t *T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)
	original := make([]byte, 32)
	copy(original, key)

	cipherSigil, err := NewChaChaPolySigil(key, nil)
	RequireNoError(t, err)

	key[0] ^= 0xFF
	AssertEqual(t, original, cipherSigil.Key())
}

func TestCryptoSigil_NewChaChaPolySigil_ShortKey_Bad(t *T) {
	cipherSigil, err := NewChaChaPolySigil([]byte("too short"), nil)
	AssertNil(t, cipherSigil)
	AssertErrorIs(t, err, InvalidKeyError)
}

func TestCryptoSigil_NewChaChaPolySigil_LongKey_Bad(t *T) {
	cipherSigil, err := NewChaChaPolySigil(make([]byte, 64), nil)
	AssertNil(t, cipherSigil)
	AssertErrorIs(t, err, InvalidKeyError)
}

func TestCryptoSigil_NewChaChaPolySigil_EmptyKey_Bad(t *T) {
	cipherSigil, err := NewChaChaPolySigil(nil, nil)
	AssertNil(t, cipherSigil)
	AssertErrorIs(t, err, InvalidKeyError)
}

func TestCryptoSigil_NewChaChaPolySigil_FixedNonceBytes_Bad(t *T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	cases := map[string][]byte{
		"non-empty": []byte("0123456789abcdef01234567"),
		"empty":     []byte{},
		"typed nil": nil,
	}
	for name, nonce := range cases {
		t.Run(name, func(t *T) {
			_, err := NewChaChaPolySigil(key, nonce)
			AssertErrorIs(t, err, InvalidNonceError)
			if err == nil {
				t.Fatal("expected invalid nonce error")
			}
			AssertContains(t, err.Error(), "fixed-nonce []byte path removed; use PreObfuscator or nil")
		})
	}
}

func TestCryptoSigil_NewChaChaPolySigil_StringNonce_Bad(t *T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	_, err := NewChaChaPolySigil(key, "fixed nonce")
	AssertErrorIs(t, err, InvalidNonceError)
	if err == nil {
		t.Fatal("expected invalid nonce error")
	}
	AssertContains(t, err.Error(), "nonce must be PreObfuscator or nil")
}

func TestCryptoSigil_NewChaChaPolySigil_CustomObfuscator_Good(t *T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	ob := &ShuffleMaskObfuscator{}
	cipherSigil, err := NewChaChaPolySigil(key, ob)
	RequireNoError(t, err)
	AssertEqual(t, ob, cipherSigil.Obfuscator())
}

func TestCryptoSigil_NewChaChaPolySigil_CustomObfuscatorNil_Good(t *T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	cipherSigil, err := NewChaChaPolySigil(key, nil)
	RequireNoError(t, err)
	_, ok := cipherSigil.Obfuscator().(*XORObfuscator)
	AssertTrue(t, ok)
}

func TestCryptoSigil_NewChaChaPolySigil_CustomObfuscator_InvalidKey_Bad(t *T) {
	obfuscator := &XORObfuscator{}
	cipherSigil, err := NewChaChaPolySigil([]byte("bad"), obfuscator)
	AssertNil(t, cipherSigil)
	AssertErrorIs(t, err, InvalidKeyError)
}

func TestCryptoSigil_ChaChaPolySigil_RoundTrip_Good(t *T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	cipherSigil, err := NewChaChaPolySigil(key, nil)
	RequireNoError(t, err)

	plaintext := []byte("consciousness does not merely avoid causing harm")
	ciphertext, err := cipherSigil.In(plaintext)
	RequireNoError(t, err)
	AssertNotEqual(t, plaintext, ciphertext)
	AssertGreater(t, len(ciphertext), len(plaintext))

	decrypted, err := cipherSigil.Out(ciphertext)
	RequireNoError(t, err)
	AssertEqual(t, plaintext, decrypted)
}

func TestCryptoSigil_ChaChaPolySigil_CustomShuffleMask_Good(t *T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	cipherSigil, err := NewChaChaPolySigil(key, &ShuffleMaskObfuscator{})
	RequireNoError(t, err)

	plaintext := []byte("shuffle mask pre-obfuscation layer")
	ciphertext, err := cipherSigil.In(plaintext)
	RequireNoError(t, err)

	decrypted, err := cipherSigil.Out(ciphertext)
	RequireNoError(t, err)
	AssertEqual(t, plaintext, decrypted)
}

func TestCryptoSigil_ChaChaPolySigil_NilData_Good(t *T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	cipherSigil, err := NewChaChaPolySigil(key, nil)
	RequireNoError(t, err)

	enc, err := cipherSigil.In(nil)
	RequireNoError(t, err)
	AssertNil(t, enc)

	dec, err := cipherSigil.Out(nil)
	RequireNoError(t, err)
	AssertNil(t, dec)
}

func TestCryptoSigil_ChaChaPolySigil_EmptyPlaintext_Good(t *T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	cipherSigil, err := NewChaChaPolySigil(key, nil)
	RequireNoError(t, err)

	ciphertext, err := cipherSigil.In([]byte{})
	RequireNoError(t, err)
	AssertNotEmpty(t, ciphertext)

	decrypted, err := cipherSigil.Out(ciphertext)
	RequireNoError(t, err)
	AssertEqual(t, []byte{}, decrypted)
}

func TestCryptoSigil_ChaChaPolySigil_DifferentCiphertextsPerCall_Good(t *T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	cipherSigil, err := NewChaChaPolySigil(key, nil)
	RequireNoError(t, err)
	cipherSigil.randomReader = &limitReader{
		data: append(bytes.Repeat([]byte{0x01}, 24), bytes.Repeat([]byte{0x02}, 24)...),
	}

	plaintext := []byte("same input")
	ct1, err := cipherSigil.In(plaintext)
	RequireNoError(t, err)
	ct2, err := cipherSigil.In(plaintext)
	RequireNoError(t, err)

	AssertNotEqual(t, ct1[:24], ct2[:24])
	AssertNotEqual(t, ct1, ct2)
}

func TestCryptoSigil_ChaChaPolySigil_NoKey_Bad(t *T) {
	cipherSigil := &ChaChaPolySigil{}

	_, err := cipherSigil.In([]byte("data"))
	AssertErrorIs(t, err, NoKeyConfiguredError)

	_, err = cipherSigil.Out([]byte("data"))
	AssertErrorIs(t, err, NoKeyConfiguredError)
}

func TestCryptoSigil_ChaChaPolySigil_WrongKey_Bad(t *T) {
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	_, _ = rand.Read(key1)
	_, _ = rand.Read(key2)

	cipherSigilOne, _ := NewChaChaPolySigil(key1, nil)
	cipherSigilTwo, _ := NewChaChaPolySigil(key2, nil)

	ciphertext, err := cipherSigilOne.In([]byte("secret"))
	RequireNoError(t, err)

	_, err = cipherSigilTwo.Out(ciphertext)
	AssertErrorIs(t, err, DecryptionFailedError)
}

func TestCryptoSigil_ChaChaPolySigil_TruncatedCiphertext_Bad(t *T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	cipherSigil, _ := NewChaChaPolySigil(key, nil)
	_, err := cipherSigil.Out([]byte("too short"))
	AssertErrorIs(t, err, CiphertextTooShortError)
}

func TestCryptoSigil_ChaChaPolySigil_TamperedCiphertext_Bad(t *T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	cipherSigil, _ := NewChaChaPolySigil(key, nil)
	ciphertext, _ := cipherSigil.In([]byte("authentic data"))

	ciphertext[30] ^= 0xFF

	_, err := cipherSigil.Out(ciphertext)
	AssertErrorIs(t, err, DecryptionFailedError)
}

type failReader struct{}

func (reader *failReader) Read([]byte) (int, error) {
	return 0, NewError("entropy source failed")
}

func TestCryptoSigil_ChaChaPolySigil_RandomReaderFailure_Bad(t *T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	cipherSigil, _ := NewChaChaPolySigil(key, nil)
	cipherSigil.randomReader = &failReader{}

	_, err := cipherSigil.In([]byte("data"))
	AssertError(t, err)
}

func TestCryptoSigil_ChaChaPolySigil_NoObfuscator_Good(t *T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	cipherSigil, _ := NewChaChaPolySigil(key, nil)
	cipherSigil.SetObfuscator(nil)

	plaintext := []byte("raw encryption without pre-obfuscation")
	ciphertext, err := cipherSigil.In(plaintext)
	RequireNoError(t, err)

	decrypted, err := cipherSigil.Out(ciphertext)
	RequireNoError(t, err)
	AssertEqual(t, plaintext, decrypted)
}

func TestCryptoSigil_NonceFromCiphertext_Good(t *T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	cipherSigil, _ := NewChaChaPolySigil(key, nil)
	ciphertext, _ := cipherSigil.In([]byte("nonce extraction test"))

	nonce, err := NonceFromCiphertext(ciphertext)
	RequireNoError(t, err)
	AssertLen(t, nonce, 24)

	AssertEqual(t, ciphertext[:24], nonce)
}

func TestCryptoSigil_NonceFromCiphertext_NonceCopied_Good(t *T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	cipherSigil, _ := NewChaChaPolySigil(key, nil)
	ciphertext, _ := cipherSigil.In([]byte("data"))

	nonce, _ := NonceFromCiphertext(ciphertext)
	original := make([]byte, len(nonce))
	copy(original, nonce)

	nonce[0] ^= 0xFF
	AssertEqual(t, original, ciphertext[:24])
}

func TestCryptoSigil_NonceFromCiphertext_TooShort_Bad(t *T) {
	nonce, err := NonceFromCiphertext([]byte("short"))
	AssertNil(t, nonce)
	AssertErrorIs(t, err, CiphertextTooShortError)
}

func TestCryptoSigil_NonceFromCiphertext_Empty_Bad(t *T) {
	nonce, err := NonceFromCiphertext(nil)
	AssertNil(t, nonce)
	AssertErrorIs(t, err, CiphertextTooShortError)
}

func TestCryptoSigil_ChaChaPolySigil_InTransmutePipeline_Good(t *T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	cipherSigil, _ := NewChaChaPolySigil(key, nil)
	hexSigil, _ := NewSigil("hex")

	chain := []Sigil{cipherSigil, hexSigil}
	plaintext := []byte("encrypt then hex encode")

	encoded, err := Transmute(plaintext, chain)
	RequireNoError(t, err)

	AssertTrue(t, isHex(encoded))

	decoded, err := Untransmute(encoded, chain)
	RequireNoError(t, err)
	AssertEqual(t, plaintext, decoded)
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

func (sigil *failSigil) In([]byte) ([]byte, error)  { return nil, NewError("fail in") }
func (sigil *failSigil) Out([]byte) ([]byte, error) { return nil, NewError("fail out") }

func TestCryptoSigil_Transmute_ErrorPropagation_Bad(t *T) {
	_, err := Transmute([]byte("data"), []Sigil{&failSigil{}})
	AssertError(t, err)
	AssertContains(t, err.Error(), "fail in")
}

func TestCryptoSigil_Untransmute_ErrorPropagation_Bad(t *T) {
	_, err := Untransmute([]byte("data"), []Sigil{&failSigil{}})
	AssertError(t, err)
	AssertContains(t, err.Error(), "fail out")
}

func TestCryptoSigil_GzipSigil_CustomOutputWriter_Good(t *T) {
	var outputBuffer bytes.Buffer
	gzipSigil := &GzipSigil{outputWriter: &outputBuffer}

	_, err := gzipSigil.In([]byte("test data"))
	RequireNoError(t, err)
	AssertGreater(t, outputBuffer.Len(), 0)
}

func TestCryptoSigil_DeriveKeyStream_ExactBlockSize_Good(t *T) {
	ob := &XORObfuscator{}
	data := make([]byte, 32)
	for i := range data {
		data[i] = byte(i)
	}
	entropy := []byte("block-boundary")

	obfuscated := ob.Obfuscate(data, entropy)
	restored := ob.Deobfuscate(obfuscated, entropy)
	AssertEqual(t, data, restored)
}

func TestCryptoSigil_ChaChaPolySigil_NilRandomReader_Good(t *T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	cipherSigil, _ := NewChaChaPolySigil(key, nil)
	cipherSigil.randomReader = nil

	ciphertext, err := cipherSigil.In([]byte("fallback reader"))
	RequireNoError(t, err)

	decrypted, err := cipherSigil.Out(ciphertext)
	RequireNoError(t, err)
	AssertEqual(t, []byte("fallback reader"), decrypted)
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
