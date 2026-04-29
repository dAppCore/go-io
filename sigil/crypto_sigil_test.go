package sigil

import (
	"crypto/rand"
	. "dappco.re/go"
	goio "io"
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

func TestCryptoSigil_ChaChaPolySigil_RoundTripGood(t *T) {
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

func TestCryptoSigil_ChaChaPolySigil_CustomShuffleMaskGood(t *T) {
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

func TestCryptoSigil_ChaChaPolySigil_NilDataGood(t *T) {
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

func TestCryptoSigil_ChaChaPolySigil_EmptyPlaintextGood(t *T) {
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

func TestCryptoSigil_ChaChaPolySigil_DifferentCiphertextsPerCallGood(t *T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	cipherSigil, err := NewChaChaPolySigil(key, nil)
	RequireNoError(t, err)
	cipherSigil.randomReader = &limitReader{
		data: append(repeatByte(0x01, 24), repeatByte(0x02, 24)...),
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

func TestCryptoSigil_ChaChaPolySigil_WrongKeyBad(t *T) {
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

func TestCryptoSigil_ChaChaPolySigil_TruncatedCiphertextBad(t *T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	cipherSigil, _ := NewChaChaPolySigil(key, nil)
	_, err := cipherSigil.Out([]byte("too short"))
	AssertErrorIs(t, err, CiphertextTooShortError)
}

func TestCryptoSigil_ChaChaPolySigil_TamperedCiphertextBad(t *T) {
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

func TestCryptoSigil_ChaChaPolySigil_RandomReaderFailureBad(t *T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	cipherSigil, _ := NewChaChaPolySigil(key, nil)
	cipherSigil.randomReader = &failReader{}

	_, err := cipherSigil.In([]byte("data"))
	AssertError(t, err)
}

func TestCryptoSigil_ChaChaPolySigil_NoObfuscatorGood(t *T) {
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

func TestCryptoSigil_ChaChaPolySigil_InTransmutePipelineGood(t *T) {
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
	outputBuffer := NewBuffer()
	gzipSigil := &GzipSigil{outputWriter: outputBuffer}

	_, err := gzipSigil.In([]byte("test data"))
	RequireNoError(t, err)
	AssertGreater(t, outputBuffer.Len(), 0)
}

func repeatByte(value byte, count int) []byte {
	out := make([]byte, count)
	for i := range out {
		out[i] = value
	}
	return out
}

func TestCryptoSigil_DeriveKeyStream_ExactBlockSizeGood(t *T) {
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

func TestCryptoSigil_ChaChaPolySigil_NilRandomReaderGood(t *T) {
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

var chachaExampleKey = []byte("0123456789abcdef0123456789abcdef")

func TestCryptoSigil_XORObfuscator_Obfuscate_Good(t *T) {
	obfuscator := &XORObfuscator{}
	got := obfuscator.Obfuscate([]byte("payload"), []byte("nonce"))
	AssertNotEqual(t, []byte("payload"), got)
	AssertLen(t, got, len("payload"))
}

func TestCryptoSigil_XORObfuscator_Obfuscate_Bad(t *T) {
	obfuscator := &XORObfuscator{}
	got := obfuscator.Obfuscate(nil, []byte("nonce"))
	AssertNil(t, got)
}

func TestCryptoSigil_XORObfuscator_Obfuscate_Ugly(t *T) {
	obfuscator := &XORObfuscator{}
	got := obfuscator.Obfuscate([]byte{}, []byte("nonce"))
	AssertEqual(t, []byte{}, got)
}

func TestCryptoSigil_XORObfuscator_Deobfuscate_Good(t *T) {
	obfuscator := &XORObfuscator{}
	encoded := obfuscator.Obfuscate([]byte("payload"), []byte("nonce"))
	got := obfuscator.Deobfuscate(encoded, []byte("nonce"))
	AssertEqual(t, []byte("payload"), got)
}

func TestCryptoSigil_XORObfuscator_Deobfuscate_Bad(t *T) {
	obfuscator := &XORObfuscator{}
	got := obfuscator.Deobfuscate(nil, []byte("nonce"))
	AssertNil(t, got)
}

func TestCryptoSigil_XORObfuscator_Deobfuscate_Ugly(t *T) {
	obfuscator := &XORObfuscator{}
	got := obfuscator.Deobfuscate([]byte{}, []byte("nonce"))
	AssertEqual(t, []byte{}, got)
}

func TestCryptoSigil_ShuffleMaskObfuscator_Obfuscate_Good(t *T) {
	obfuscator := &ShuffleMaskObfuscator{}
	got := obfuscator.Obfuscate([]byte("payload"), []byte("nonce"))
	AssertNotEqual(t, []byte("payload"), got)
	AssertLen(t, got, len("payload"))
}

func TestCryptoSigil_ShuffleMaskObfuscator_Obfuscate_Bad(t *T) {
	obfuscator := &ShuffleMaskObfuscator{}
	got := obfuscator.Obfuscate(nil, []byte("nonce"))
	AssertNil(t, got)
}

func TestCryptoSigil_ShuffleMaskObfuscator_Obfuscate_Ugly(t *T) {
	obfuscator := &ShuffleMaskObfuscator{}
	got := obfuscator.Obfuscate([]byte{1}, []byte("nonce"))
	AssertLen(t, got, 1)
}

func TestCryptoSigil_ShuffleMaskObfuscator_Deobfuscate_Good(t *T) {
	obfuscator := &ShuffleMaskObfuscator{}
	encoded := obfuscator.Obfuscate([]byte("payload"), []byte("nonce"))
	got := obfuscator.Deobfuscate(encoded, []byte("nonce"))
	AssertEqual(t, []byte("payload"), got)
}

func TestCryptoSigil_ShuffleMaskObfuscator_Deobfuscate_Bad(t *T) {
	obfuscator := &ShuffleMaskObfuscator{}
	got := obfuscator.Deobfuscate(nil, []byte("nonce"))
	AssertNil(t, got)
}

func TestCryptoSigil_ShuffleMaskObfuscator_Deobfuscate_Ugly(t *T) {
	obfuscator := &ShuffleMaskObfuscator{}
	encoded := obfuscator.Obfuscate([]byte{1}, []byte("nonce"))
	got := obfuscator.Deobfuscate(encoded, []byte("nonce"))
	AssertEqual(t, []byte{1}, got)
}

func TestCryptoSigil_NewChaChaPolySigil_Bad(t *T) {
	sigilValue, err := NewChaChaPolySigil([]byte("short"), nil)
	AssertErrorIs(t, err, InvalidKeyError)
	AssertNil(t, sigilValue)
}

func TestCryptoSigil_NewChaChaPolySigil_Ugly(t *T) {
	sigilValue, err := NewChaChaPolySigil(chachaExampleKey, &ShuffleMaskObfuscator{})
	AssertNoError(t, err)
	AssertNotNil(t, sigilValue.Obfuscator())
}

func TestCryptoSigil_ChaChaPolySigil_Key_Good(t *T) {
	sigilValue, err := NewChaChaPolySigil(chachaExampleKey, nil)
	RequireNoError(t, err)
	got := sigilValue.Key()
	AssertEqual(t, chachaExampleKey, got)
}

func TestCryptoSigil_ChaChaPolySigil_Key_Bad(t *T) {
	sigilValue := &ChaChaPolySigil{}
	got := sigilValue.Key()
	AssertEmpty(t, got)
}

func TestCryptoSigil_ChaChaPolySigil_Key_Ugly(t *T) {
	sigilValue, err := NewChaChaPolySigil(chachaExampleKey, nil)
	RequireNoError(t, err)
	got := sigilValue.Key()
	got[0] ^= 0xff
	AssertEqual(t, chachaExampleKey, sigilValue.Key())
}

func TestCryptoSigil_ChaChaPolySigil_Nonce_Good(t *T) {
	sigilValue, err := NewChaChaPolySigil(chachaExampleKey, nil)
	RequireNoError(t, err)
	got := sigilValue.Nonce()
	AssertNil(t, got)
}

func TestCryptoSigil_ChaChaPolySigil_Nonce_Bad(t *T) {
	sigilValue := &ChaChaPolySigil{}
	got := sigilValue.Nonce()
	AssertNil(t, got)
}

func TestCryptoSigil_ChaChaPolySigil_Nonce_Ugly(t *T) {
	sigilValue, err := NewChaChaPolySigil(chachaExampleKey, &ShuffleMaskObfuscator{})
	RequireNoError(t, err)
	got := sigilValue.Nonce()
	AssertNil(t, got)
}

func TestCryptoSigil_ChaChaPolySigil_Obfuscator_Good(t *T) {
	sigilValue, err := NewChaChaPolySigil(chachaExampleKey, nil)
	RequireNoError(t, err)
	got := sigilValue.Obfuscator()
	AssertNotNil(t, got)
}

func TestCryptoSigil_ChaChaPolySigil_Obfuscator_Bad(t *T) {
	sigilValue := &ChaChaPolySigil{}
	got := sigilValue.Obfuscator()
	AssertNil(t, got)
}

func TestCryptoSigil_ChaChaPolySigil_Obfuscator_Ugly(t *T) {
	obfuscator := &ShuffleMaskObfuscator{}
	sigilValue, err := NewChaChaPolySigil(chachaExampleKey, obfuscator)
	RequireNoError(t, err)
	AssertSame(t, obfuscator, sigilValue.Obfuscator())
}

func TestCryptoSigil_ChaChaPolySigil_SetObfuscator_Good(t *T) {
	sigilValue, err := NewChaChaPolySigil(chachaExampleKey, nil)
	RequireNoError(t, err)
	obfuscator := &ShuffleMaskObfuscator{}
	sigilValue.SetObfuscator(obfuscator)
	AssertSame(t, obfuscator, sigilValue.Obfuscator())
}

func TestCryptoSigil_ChaChaPolySigil_SetObfuscator_Bad(t *T) {
	sigilValue, err := NewChaChaPolySigil(chachaExampleKey, nil)
	RequireNoError(t, err)
	sigilValue.SetObfuscator(nil)
	AssertNil(t, sigilValue.Obfuscator())
}

func TestCryptoSigil_ChaChaPolySigil_SetObfuscator_Ugly(t *T) {
	sigilValue, err := NewChaChaPolySigil(chachaExampleKey, &ShuffleMaskObfuscator{})
	RequireNoError(t, err)
	sigilValue.SetObfuscator(&XORObfuscator{})
	AssertNotNil(t, sigilValue.Obfuscator())
}

func TestCryptoSigil_ChaChaPolySigil_In_Good(t *T) {
	sigilValue, err := NewChaChaPolySigil(chachaExampleKey, nil)
	RequireNoError(t, err)
	got, err := sigilValue.In([]byte("payload"))
	AssertNoError(t, err)
	AssertNotEmpty(t, got)
}

func TestCryptoSigil_ChaChaPolySigil_In_Bad(t *T) {
	sigilValue := &ChaChaPolySigil{}
	got, err := sigilValue.In([]byte("payload"))
	AssertErrorIs(t, err, NoKeyConfiguredError)
	AssertNil(t, got)
}

func TestCryptoSigil_ChaChaPolySigil_In_Ugly(t *T) {
	sigilValue, err := NewChaChaPolySigil(chachaExampleKey, nil)
	RequireNoError(t, err)
	got, err := sigilValue.In(nil)
	AssertNoError(t, err)
	AssertNil(t, got)
}

func TestCryptoSigil_ChaChaPolySigil_Out_Good(t *T) {
	sigilValue, err := NewChaChaPolySigil(chachaExampleKey, nil)
	RequireNoError(t, err)
	ciphertext, err := sigilValue.In([]byte("payload"))
	RequireNoError(t, err)
	got, err := sigilValue.Out(ciphertext)
	AssertNoError(t, err)
	AssertEqual(t, []byte("payload"), got)
}

func TestCryptoSigil_ChaChaPolySigil_Out_Bad(t *T) {
	sigilValue := &ChaChaPolySigil{}
	got, err := sigilValue.Out([]byte("ciphertext"))
	AssertErrorIs(t, err, NoKeyConfiguredError)
	AssertNil(t, got)
}

func TestCryptoSigil_ChaChaPolySigil_Out_Ugly(t *T) {
	sigilValue, err := NewChaChaPolySigil(chachaExampleKey, nil)
	RequireNoError(t, err)
	got, err := sigilValue.Out(nil)
	AssertNoError(t, err)
	AssertNil(t, got)
}

func TestCryptoSigil_NonceFromCiphertext_Bad(t *T) {
	nonce, err := NonceFromCiphertext([]byte("short"))
	AssertErrorIs(t, err, CiphertextTooShortError)
	AssertNil(t, nonce)
}

func TestCryptoSigil_NonceFromCiphertext_Ugly(t *T) {
	nonce, err := NonceFromCiphertext(make([]byte, 24))
	AssertNoError(t, err)
	AssertLen(t, nonce, 24)
}
