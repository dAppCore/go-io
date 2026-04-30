package sigil

import (
	"compress/gzip" // AX-6-exception: gzip transport encoding has no core equivalent.
	"crypto"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"io/fs" // AX-6-exception: fs sentinel errors have no core equivalent.

	core "dappco.re/go"
	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/blake2s"
	"golang.org/x/crypto/md4"       //nolint:staticcheck // Supported for legacy hash-name compatibility.
	"golang.org/x/crypto/ripemd160" //nolint:staticcheck // Supported for legacy hash-name compatibility.
	"golang.org/x/crypto/sha3"
)

const (
	opGzipOut          = "sigil.GzipSigil.Out"
	errReadGzipPayload = "read gzip payload"
)

// Example: reverseSigil, _ := sigil.NewSigil("reverse")
type ReverseSigil struct{}

func (sigil *ReverseSigil) In(data []byte) ([]byte, error) {
	if data == nil {
		return nil, nil
	}
	reversed := make([]byte, len(data))
	for i, j := 0, len(data)-1; i < len(data); i, j = i+1, j-1 {
		reversed[i] = data[j]
	}
	return reversed, nil
}

func (sigil *ReverseSigil) Out(data []byte) ([]byte, error) {
	return sigil.In(data)
}

// Example: hexSigil, _ := sigil.NewSigil("hex")
type HexSigil struct{}

func (sigil *HexSigil) In(data []byte) ([]byte, error) {
	if data == nil {
		return nil, nil
	}
	encodedBytes := make([]byte, hex.EncodedLen(len(data)))
	hex.Encode(encodedBytes, data)
	return encodedBytes, nil
}

func (sigil *HexSigil) Out(data []byte) ([]byte, error) {
	if data == nil {
		return nil, nil
	}
	decodedBytes := make([]byte, hex.DecodedLen(len(data)))
	_, err := hex.Decode(decodedBytes, data)
	return decodedBytes, err
}

// Example: base64Sigil, _ := sigil.NewSigil("base64")
type Base64Sigil struct{}

func (sigil *Base64Sigil) In(data []byte) ([]byte, error) {
	if data == nil {
		return nil, nil
	}
	encodedBytes := make([]byte, base64.StdEncoding.EncodedLen(len(data)))
	base64.StdEncoding.Encode(encodedBytes, data)
	return encodedBytes, nil
}

func (sigil *Base64Sigil) Out(data []byte) ([]byte, error) {
	if data == nil {
		return nil, nil
	}
	decodedBytes := make([]byte, base64.StdEncoding.DecodedLen(len(data)))
	decodedCount, err := base64.StdEncoding.Decode(decodedBytes, data)
	return decodedBytes[:decodedCount], err
}

// Example: gzipSigil, _ := sigil.NewSigil("gzip")
type GzipSigil struct {
	outputWriter sigilWriter
}

type sigilWriter interface {
	Write([]byte) (int, error)
}

type sigilHash interface {
	sigilWriter
	Sum([]byte) []byte
}

// AX-6-exception: core.NewBuffer is unavailable in the pinned core module; this is
// the minimal intrinsic writer needed by compress/gzip.
type sigilBuffer struct {
	data []byte
}

func (buffer *sigilBuffer) Write(data []byte) (int, error) {
	buffer.data = append(buffer.data, data...)
	return len(data), nil
}

func (buffer *sigilBuffer) Bytes() []byte {
	return buffer.data
}

func (sigil *GzipSigil) In(data []byte) ([]byte, error) {
	if data == nil {
		return nil, nil
	}
	var buffer sigilBuffer
	outputWriter := sigil.outputWriter
	if outputWriter == nil {
		outputWriter = &buffer
	}
	gzipWriter := gzip.NewWriter(outputWriter)
	if _, err := gzipWriter.Write(data); err != nil {
		return nil, core.E("sigil.GzipSigil.In", "write gzip payload", err)
	}
	if err := gzipWriter.Close(); err != nil {
		return nil, core.E("sigil.GzipSigil.In", "close gzip writer", err)
	}
	// When a custom outputWriter was supplied the caller owns the bytes; return nil so the
	// pipeline does not propagate a stale empty-buffer value.
	if sigil.outputWriter != nil {
		return nil, nil
	}
	return buffer.Bytes(), nil
}

func (sigil *GzipSigil) Out(data []byte) ([]byte, error) {
	if data == nil {
		return nil, nil
	}
	gzipReader, err := gzip.NewReader(core.NewReader(string(data)))
	if err != nil {
		return nil, core.E(opGzipOut, "open gzip reader", err)
	}
	defer closeGzipReader(gzipReader)
	out := core.ReadAll(gzipReader)
	if !out.OK {
		if err, ok := out.Value.(error); ok {
			return nil, core.E(opGzipOut, errReadGzipPayload, err)
		}
		return nil, core.E(opGzipOut, errReadGzipPayload, fs.ErrInvalid)
	}
	return []byte(out.Value.(string)), nil
}

func closeGzipReader(reader interface{ Close() error }) {
	if err := reader.Close(); err != nil {
		core.Warn("gzip reader close failed", "err", err)
	}
}

// Example: jsonSigil := &sigil.JSONSigil{Indent: true}
type JSONSigil struct{ Indent bool }

func (sigil *JSONSigil) In(data []byte) ([]byte, error) {
	if data == nil {
		return nil, nil
	}

	var decoded any
	result := core.JSONUnmarshal(data, &decoded)
	if !result.OK {
		if err, ok := result.Value.(error); ok {
			return nil, core.E("sigil.JSONSigil.In", "decode json", err)
		}
		return nil, core.E("sigil.JSONSigil.In", "decode json", fs.ErrInvalid)
	}

	compact := core.JSONMarshalString(decoded)
	if sigil.Indent {
		return []byte(indentJSON(compact)), nil
	}
	return []byte(compact), nil
}

func (sigil *JSONSigil) Out(data []byte) ([]byte, error) {
	return data, nil
}

// Example: hashSigil := sigil.NewHashSigil(crypto.SHA256)
type HashSigil struct {
	Hash crypto.Hash
}

// Example: hashSigil := sigil.NewHashSigil(crypto.SHA256)
// Example: digest, _ := hashSigil.In([]byte("payload"))
func NewHashSigil(hashAlgorithm crypto.Hash) *HashSigil {
	return &HashSigil{Hash: hashAlgorithm}
}

func (sigil *HashSigil) In(data []byte) ([]byte, error) {
	var hasher sigilHash
	var err error
	switch sigil.Hash {
	case crypto.MD4:
		hasher = md4.New()
	case crypto.MD5:
		hasher = md5.New()
	case crypto.SHA1:
		hasher = sha1.New()
	case crypto.SHA224:
		hasher = sha256.New224()
	case crypto.SHA256:
		hasher = sha256.New()
	case crypto.SHA384:
		hasher = sha512.New384()
	case crypto.SHA512:
		hasher = sha512.New()
	case crypto.RIPEMD160:
		hasher = ripemd160.New()
	case crypto.SHA3_224:
		hasher = sha3.New224()
	case crypto.SHA3_256:
		hasher = sha3.New256()
	case crypto.SHA3_384:
		hasher = sha3.New384()
	case crypto.SHA3_512:
		hasher = sha3.New512()
	case crypto.SHA512_224:
		hasher = sha512.New512_224()
	case crypto.SHA512_256:
		hasher = sha512.New512_256()
	case crypto.BLAKE2s_256:
		hasher, err = blake2s.New256(nil)
	case crypto.BLAKE2b_256:
		hasher, err = blake2b.New256(nil)
	case crypto.BLAKE2b_384:
		hasher, err = blake2b.New384(nil)
	case crypto.BLAKE2b_512:
		hasher, err = blake2b.New512(nil)
	default:
		return nil, core.E("sigil.HashSigil.In", "hash algorithm not available", fs.ErrInvalid)
	}
	if err != nil {
		return nil, core.E("sigil.HashSigil.In", "create hash", err)
	}

	if _, err := hasher.Write(data); err != nil {
		return nil, core.E("sigil.HashSigil.In", "write hash input", err)
	}
	return hasher.Sum(nil), nil
}

func (sigil *HashSigil) Out(data []byte) ([]byte, error) {
	return data, nil
}

// NewSigil constructs sigils that do not require caller-provided construction
// material. ChaCha20-Poly1305 requires key material at construction; use
// NewChaChaPolySigil instead.
//
// Example: hexSigil, _ := sigil.NewSigil("hex")
// Example: gzipSigil, _ := sigil.NewSigil("gzip")
// Example: transformed, _ := sigil.Transmute([]byte("payload"), []sigil.Sigil{hexSigil, gzipSigil})
func NewSigil(sigilName string) (Sigil, error) {
	if sigilName == "chacha20poly1305" {
		return nil, core.E("sigil.NewSigil", "chacha20poly1305 scheme requires key material; use NewChaChaPolySigil", fs.ErrInvalid)
	}

	switch sigilName {
	case "reverse":
		return &ReverseSigil{}, nil
	case "hex":
		return &HexSigil{}, nil
	case "base64":
		return &Base64Sigil{}, nil
	case "gzip":
		return &GzipSigil{}, nil
	case "json":
		return &JSONSigil{Indent: false}, nil
	case "json-indent":
		return &JSONSigil{Indent: true}, nil
	case "md4":
		return NewHashSigil(crypto.MD4), nil
	case "md5":
		return NewHashSigil(crypto.MD5), nil
	case "sha1":
		return NewHashSigil(crypto.SHA1), nil
	case "sha224":
		return NewHashSigil(crypto.SHA224), nil
	case "sha256":
		return NewHashSigil(crypto.SHA256), nil
	case "sha384":
		return NewHashSigil(crypto.SHA384), nil
	case "sha512":
		return NewHashSigil(crypto.SHA512), nil
	case "ripemd160":
		return NewHashSigil(crypto.RIPEMD160), nil
	case "sha3-224":
		return NewHashSigil(crypto.SHA3_224), nil
	case "sha3-256":
		return NewHashSigil(crypto.SHA3_256), nil
	case "sha3-384":
		return NewHashSigil(crypto.SHA3_384), nil
	case "sha3-512":
		return NewHashSigil(crypto.SHA3_512), nil
	case "sha512-224":
		return NewHashSigil(crypto.SHA512_224), nil
	case "sha512-256":
		return NewHashSigil(crypto.SHA512_256), nil
	case "blake2s-256":
		return NewHashSigil(crypto.BLAKE2s_256), nil
	case "blake2b-256":
		return NewHashSigil(crypto.BLAKE2b_256), nil
	case "blake2b-384":
		return NewHashSigil(crypto.BLAKE2b_384), nil
	case "blake2b-512":
		return NewHashSigil(crypto.BLAKE2b_512), nil
	default:
		return nil, core.E("sigil.NewSigil", core.Concat("unknown sigil name: ", sigilName), fs.ErrInvalid)
	}
}

func indentJSON(compact string) string {
	if compact == "" {
		return ""
	}

	builder := core.NewBuilder()
	indent := 0
	inString := false
	escaped := false

	for i := 0; i < len(compact); i++ {
		ch := compact[i]
		if inString {
			inString, escaped = writeJSONStringByte(builder, ch, escaped)
			continue
		}

		indent, inString = writeJSONStructuralByte(builder, compact, i, indent)
	}

	return builder.String()
}

type jsonIndentWriter interface {
	WriteByte(byte) error
	WriteString(string) (int, error)
}

func writeJSONByte(builder jsonIndentWriter, ch byte) {
	if err := builder.WriteByte(ch); err != nil {
		core.Warn("json indent byte write failed", "err", err)
	}
}

func writeJSONString(builder jsonIndentWriter, text string) {
	if _, err := builder.WriteString(text); err != nil {
		core.Warn("json indent string write failed", "err", err)
	}
}

func writeJSONIndent(builder jsonIndentWriter, level int) {
	for i := 0; i < level; i++ {
		writeJSONString(builder, "  ")
	}
}

func writeJSONStringByte(builder jsonIndentWriter, ch byte, escaped bool) (bool, bool) {
	writeJSONByte(builder, ch)
	if escaped {
		return true, false
	}
	if ch == '\\' {
		return true, true
	}
	return ch != '"', false
}

func writeJSONStructuralByte(builder jsonIndentWriter, compact string, index, indent int) (int, bool) {
	ch := compact[index]
	switch ch {
	case '"':
		writeJSONByte(builder, ch)
		return indent, true
	case '{', '[':
		writeJSONByte(builder, ch)
		return writeJSONOpenContainer(builder, compact, index, indent), false
	case '}', ']':
		indent = writeJSONCloseContainer(builder, compact, index, indent)
		writeJSONByte(builder, ch)
	case ',':
		writeJSONByte(builder, ch)
		writeJSONByte(builder, '\n')
		writeJSONIndent(builder, indent)
	case ':':
		writeJSONString(builder, ": ")
	default:
		writeJSONByte(builder, ch)
	}
	return indent, false
}

func writeJSONOpenContainer(builder jsonIndentWriter, compact string, index, indent int) int {
	if index+1 >= len(compact) || compact[index+1] == '}' || compact[index+1] == ']' {
		return indent
	}
	indent++
	writeJSONByte(builder, '\n')
	writeJSONIndent(builder, indent)
	return indent
}

func writeJSONCloseContainer(builder jsonIndentWriter, compact string, index, indent int) int {
	if index == 0 || compact[index-1] == '{' || compact[index-1] == '[' {
		return indent
	}
	indent--
	writeJSONByte(builder, '\n')
	writeJSONIndent(builder, indent)
	return indent
}
