package sigil

import (
	"crypto"
	core "dappco.re/go"
)

func TestSigils_ReverseSigil_In_Good(t *core.T) {
	sigilValue := &ReverseSigil{}
	got, err := sigilValue.In([]byte("abc"))
	core.AssertNoError(t, err)
	core.AssertEqual(t, []byte("cba"), got)
}

func TestSigils_ReverseSigil_In_Bad(t *core.T) {
	sigilValue := &ReverseSigil{}
	got, err := sigilValue.In(nil)
	core.AssertNoError(t, err)
	core.AssertNil(t, got)
}

func TestSigils_ReverseSigil_In_Ugly(t *core.T) {
	sigilValue := &ReverseSigil{}
	got, err := sigilValue.In([]byte(""))
	core.AssertNoError(t, err)
	core.AssertEqual(t, []byte{}, got)
}

func TestSigils_ReverseSigil_Out_Good(t *core.T) {
	sigilValue := &ReverseSigil{}
	got, err := sigilValue.Out([]byte("cba"))
	core.AssertNoError(t, err)
	core.AssertEqual(t, []byte("abc"), got)
}

func TestSigils_ReverseSigil_Out_Bad(t *core.T) {
	sigilValue := &ReverseSigil{}
	got, err := sigilValue.Out(nil)
	core.AssertNoError(t, err)
	core.AssertNil(t, got)
}

func TestSigils_ReverseSigil_Out_Ugly(t *core.T) {
	sigilValue := &ReverseSigil{}
	got, err := sigilValue.Out([]byte(""))
	core.AssertNoError(t, err)
	core.AssertEqual(t, []byte{}, got)
}

func TestSigils_HexSigil_In_Good(t *core.T) {
	sigilValue := &HexSigil{}
	got, err := sigilValue.In([]byte("hi"))
	core.AssertNoError(t, err)
	core.AssertEqual(t, []byte("6869"), got)
}

func TestSigils_HexSigil_In_Bad(t *core.T) {
	sigilValue := &HexSigil{}
	got, err := sigilValue.In(nil)
	core.AssertNoError(t, err)
	core.AssertNil(t, got)
}

func TestSigils_HexSigil_In_Ugly(t *core.T) {
	sigilValue := &HexSigil{}
	got, err := sigilValue.In([]byte{})
	core.AssertNoError(t, err)
	core.AssertEqual(t, []byte{}, got)
}

func TestSigils_HexSigil_Out_Good(t *core.T) {
	sigilValue := &HexSigil{}
	got, err := sigilValue.Out([]byte("6869"))
	core.AssertNoError(t, err)
	core.AssertEqual(t, []byte("hi"), got)
}

func TestSigils_HexSigil_Out_Bad(t *core.T) {
	sigilValue := &HexSigil{}
	got, err := sigilValue.Out([]byte("zz"))
	core.AssertError(t, err)
	core.AssertEqual(t, []byte{0}, got)
}

func TestSigils_HexSigil_Out_Ugly(t *core.T) {
	sigilValue := &HexSigil{}
	got, err := sigilValue.Out(nil)
	core.AssertNoError(t, err)
	core.AssertNil(t, got)
}

func TestSigils_Base64Sigil_In_Good(t *core.T) {
	sigilValue := &Base64Sigil{}
	got, err := sigilValue.In([]byte("hi"))
	core.AssertNoError(t, err)
	core.AssertEqual(t, []byte("aGk="), got)
}

func TestSigils_Base64Sigil_In_Bad(t *core.T) {
	sigilValue := &Base64Sigil{}
	got, err := sigilValue.In(nil)
	core.AssertNoError(t, err)
	core.AssertNil(t, got)
}

func TestSigils_Base64Sigil_In_Ugly(t *core.T) {
	sigilValue := &Base64Sigil{}
	got, err := sigilValue.In([]byte{})
	core.AssertNoError(t, err)
	core.AssertEqual(t, []byte{}, got)
}

func TestSigils_Base64Sigil_Out_Good(t *core.T) {
	sigilValue := &Base64Sigil{}
	got, err := sigilValue.Out([]byte("aGk="))
	core.AssertNoError(t, err)
	core.AssertEqual(t, []byte("hi"), got)
}

func TestSigils_Base64Sigil_Out_Bad(t *core.T) {
	sigilValue := &Base64Sigil{}
	got, err := sigilValue.Out([]byte("!!!"))
	core.AssertError(t, err)
	core.AssertEmpty(t, got)
}

func TestSigils_Base64Sigil_Out_Ugly(t *core.T) {
	sigilValue := &Base64Sigil{}
	got, err := sigilValue.Out(nil)
	core.AssertNoError(t, err)
	core.AssertNil(t, got)
}

func TestSigils_GzipSigil_In_Good(t *core.T) {
	sigilValue := &GzipSigil{}
	got, err := sigilValue.In([]byte("payload"))
	core.AssertNoError(t, err)
	core.AssertNotEmpty(t, got)
}

func TestSigils_GzipSigil_In_Bad(t *core.T) {
	sigilValue := &GzipSigil{}
	got, err := sigilValue.In(nil)
	core.AssertNoError(t, err)
	core.AssertNil(t, got)
}

func TestSigils_GzipSigil_In_Ugly(t *core.T) {
	buffer := &sigilBuffer{}
	sigilValue := &GzipSigil{outputWriter: buffer}
	got, err := sigilValue.In([]byte("payload"))
	core.AssertNoError(t, err)
	core.AssertNil(t, got)
	core.AssertNotEmpty(t, buffer.Bytes())
}

func TestSigils_GzipSigil_Out_Good(t *core.T) {
	sigilValue := &GzipSigil{}
	compressed, err := sigilValue.In([]byte("payload"))
	core.RequireNoError(t, err)
	got, err := sigilValue.Out(compressed)
	core.AssertNoError(t, err)
	core.AssertEqual(t, []byte("payload"), got)
}

func TestSigils_GzipSigil_Out_Bad(t *core.T) {
	sigilValue := &GzipSigil{}
	got, err := sigilValue.Out([]byte("not gzip"))
	core.AssertError(t, err)
	core.AssertNil(t, got)
}

func TestSigils_GzipSigil_Out_Ugly(t *core.T) {
	sigilValue := &GzipSigil{}
	got, err := sigilValue.Out(nil)
	core.AssertNoError(t, err)
	core.AssertNil(t, got)
}

func TestSigils_Buffer_Write_Good(t *core.T) {
	buffer := &sigilBuffer{}
	count, err := buffer.Write([]byte("payload"))
	core.AssertNoError(t, err)
	core.AssertEqual(t, len("payload"), count)
}

func TestSigils_Buffer_Write_Bad(t *core.T) {
	buffer := &sigilBuffer{}
	count, err := buffer.Write(nil)
	core.AssertNoError(t, err)
	core.AssertEqual(t, 0, count)
}

func TestSigils_Buffer_Write_Ugly(t *core.T) {
	buffer := &sigilBuffer{data: []byte("a")}
	count, err := buffer.Write([]byte("b"))
	core.AssertNoError(t, err)
	core.AssertEqual(t, 1, count)
}

func TestSigils_Buffer_Bytes_Good(t *core.T) {
	buffer := &sigilBuffer{data: []byte("payload")}
	got := buffer.Bytes()
	core.AssertEqual(t, []byte("payload"), got)
}

func TestSigils_Buffer_Bytes_Bad(t *core.T) {
	buffer := &sigilBuffer{}
	got := buffer.Bytes()
	core.AssertNil(t, got)
}

func TestSigils_Buffer_Bytes_Ugly(t *core.T) {
	buffer := &sigilBuffer{data: []byte{}}
	got := buffer.Bytes()
	core.AssertEqual(t, []byte{}, got)
}

func TestSigils_JSONSigil_In_Good(t *core.T) {
	sigilValue := &JSONSigil{}
	got, err := sigilValue.In([]byte(`{ "key" : "value" }`))
	core.AssertNoError(t, err)
	core.AssertEqual(t, []byte(`{"key":"value"}`), got)
}

func TestSigils_JSONSigil_In_Bad(t *core.T) {
	sigilValue := &JSONSigil{}
	got, err := sigilValue.In([]byte("not json"))
	core.AssertError(t, err)
	core.AssertNil(t, got)
}

func TestSigils_JSONSigil_In_Ugly(t *core.T) {
	sigilValue := &JSONSigil{Indent: true}
	got, err := sigilValue.In([]byte(`{"key":"value"}`))
	core.AssertNoError(t, err)
	core.AssertContains(t, string(got), "\n")
}

func TestSigils_JSONSigil_Out_Good(t *core.T) {
	sigilValue := &JSONSigil{}
	got, err := sigilValue.Out([]byte(`{"key":"value"}`))
	core.AssertNoError(t, err)
	core.AssertEqual(t, []byte(`{"key":"value"}`), got)
}

func TestSigils_JSONSigil_Out_Bad(t *core.T) {
	sigilValue := &JSONSigil{}
	got, err := sigilValue.Out(nil)
	core.AssertNoError(t, err)
	core.AssertNil(t, got)
}

func TestSigils_JSONSigil_Out_Ugly(t *core.T) {
	sigilValue := &JSONSigil{Indent: true}
	got, err := sigilValue.Out([]byte("not json"))
	core.AssertNoError(t, err)
	core.AssertEqual(t, []byte("not json"), got)
}

func TestSigils_NewHashSigil_Good(t *core.T) {
	sigilValue := NewHashSigil(crypto.SHA256)
	core.AssertNotNil(t, sigilValue)
	core.AssertEqual(t, crypto.SHA256, sigilValue.Hash)
}

func TestSigils_NewHashSigil_Bad(t *core.T) {
	sigilValue := NewHashSigil(crypto.Hash(0))
	_, err := sigilValue.In([]byte("payload"))
	core.AssertError(t, err)
}

func TestSigils_NewHashSigil_Ugly(t *core.T) {
	sigilValue := NewHashSigil(crypto.MD5)
	got, err := sigilValue.In([]byte{})
	core.AssertNoError(t, err)
	core.AssertLen(t, got, 16)
}

func TestSigils_HashSigil_In_Good(t *core.T) {
	sigilValue := &HashSigil{Hash: crypto.SHA256}
	got, err := sigilValue.In([]byte("payload"))
	core.AssertNoError(t, err)
	core.AssertLen(t, got, 32)
}

func TestSigils_HashSigil_In_Bad(t *core.T) {
	sigilValue := &HashSigil{Hash: crypto.Hash(0)}
	got, err := sigilValue.In([]byte("payload"))
	core.AssertError(t, err)
	core.AssertNil(t, got)
}

func TestSigils_HashSigil_In_Ugly(t *core.T) {
	sigilValue := &HashSigil{Hash: crypto.SHA512}
	got, err := sigilValue.In(nil)
	core.AssertNoError(t, err)
	core.AssertLen(t, got, 64)
}

func TestSigils_HashSigil_Out_Good(t *core.T) {
	sigilValue := &HashSigil{Hash: crypto.SHA256}
	got, err := sigilValue.Out([]byte("digest"))
	core.AssertNoError(t, err)
	core.AssertEqual(t, []byte("digest"), got)
}

func TestSigils_HashSigil_Out_Bad(t *core.T) {
	sigilValue := &HashSigil{}
	got, err := sigilValue.Out(nil)
	core.AssertNoError(t, err)
	core.AssertNil(t, got)
}

func TestSigils_HashSigil_Out_Ugly(t *core.T) {
	sigilValue := &HashSigil{Hash: crypto.MD5}
	got, err := sigilValue.Out([]byte{})
	core.AssertNoError(t, err)
	core.AssertEqual(t, []byte{}, got)
}

func TestSigils_NewSigil_Good(t *core.T) {
	sigilValue, err := NewSigil("hex")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, sigilValue)
}

func TestSigils_NewSigil_Bad(t *core.T) {
	sigilValue, err := NewSigil("missing")
	core.AssertError(t, err)
	core.AssertNil(t, sigilValue)
}

func TestSigils_NewSigil_Ugly(t *core.T) {
	sigilValue, err := NewSigil("chacha20poly1305")
	core.AssertError(t, err)
	core.AssertNil(t, sigilValue)
}
