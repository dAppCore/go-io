package sigil

import (
	"bytes"
	"compress/gzip"
	"crypto"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	goio "io"

	core "dappco.re/go/core"
	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/blake2s"
	"golang.org/x/crypto/md4"
	"golang.org/x/crypto/ripemd160"
	"golang.org/x/crypto/sha3"
)

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

type HexSigil struct{}

func (sigil *HexSigil) In(data []byte) ([]byte, error) {
	if data == nil {
		return nil, nil
	}
	dst := make([]byte, hex.EncodedLen(len(data)))
	hex.Encode(dst, data)
	return dst, nil
}

func (sigil *HexSigil) Out(data []byte) ([]byte, error) {
	if data == nil {
		return nil, nil
	}
	dst := make([]byte, hex.DecodedLen(len(data)))
	_, err := hex.Decode(dst, data)
	return dst, err
}

type Base64Sigil struct{}

func (sigil *Base64Sigil) In(data []byte) ([]byte, error) {
	if data == nil {
		return nil, nil
	}
	dst := make([]byte, base64.StdEncoding.EncodedLen(len(data)))
	base64.StdEncoding.Encode(dst, data)
	return dst, nil
}

func (sigil *Base64Sigil) Out(data []byte) ([]byte, error) {
	if data == nil {
		return nil, nil
	}
	dst := make([]byte, base64.StdEncoding.DecodedLen(len(data)))
	n, err := base64.StdEncoding.Decode(dst, data)
	return dst[:n], err
}

type GzipSigil struct {
	outputWriter goio.Writer
}

func (sigil *GzipSigil) In(data []byte) ([]byte, error) {
	if data == nil {
		return nil, nil
	}
	var b bytes.Buffer
	outputWriter := sigil.outputWriter
	if outputWriter == nil {
		outputWriter = &b
	}
	gzipWriter := gzip.NewWriter(outputWriter)
	if _, err := gzipWriter.Write(data); err != nil {
		return nil, core.E("sigil.GzipSigil.In", "write gzip payload", err)
	}
	if err := gzipWriter.Close(); err != nil {
		return nil, core.E("sigil.GzipSigil.In", "close gzip writer", err)
	}
	return b.Bytes(), nil
}

func (sigil *GzipSigil) Out(data []byte) ([]byte, error) {
	if data == nil {
		return nil, nil
	}
	gzipReader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, core.E("sigil.GzipSigil.Out", "open gzip reader", err)
	}
	defer gzipReader.Close()
	out, err := goio.ReadAll(gzipReader)
	if err != nil {
		return nil, core.E("sigil.GzipSigil.Out", "read gzip payload", err)
	}
	return out, nil
}

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
		return nil, core.E("sigil.JSONSigil.In", "decode json", nil)
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

type HashSigil struct {
	Hash crypto.Hash
}

// Example: hashSigil := sigil.NewHashSigil(crypto.SHA256)
// Example: digest, _ := hashSigil.In([]byte("payload"))
func NewHashSigil(h crypto.Hash) *HashSigil {
	return &HashSigil{Hash: h}
}

func (sigil *HashSigil) In(data []byte) ([]byte, error) {
	var hasher goio.Writer
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
		hasher, _ = blake2s.New256(nil)
	case crypto.BLAKE2b_256:
		hasher, _ = blake2b.New256(nil)
	case crypto.BLAKE2b_384:
		hasher, _ = blake2b.New384(nil)
	case crypto.BLAKE2b_512:
		hasher, _ = blake2b.New512(nil)
	default:
		return nil, core.E("sigil.HashSigil.In", "hash algorithm not available", nil)
	}

	hasher.Write(data)
	return hasher.(interface{ Sum([]byte) []byte }).Sum(nil), nil
}

func (sigil *HashSigil) Out(data []byte) ([]byte, error) {
	return data, nil
}

// Example: hexSigil, _ := sigil.NewSigil("hex")
// Example: gzipSigil, _ := sigil.NewSigil("gzip")
// Example: transformed, _ := sigil.Transmute([]byte("payload"), []sigil.Sigil{hexSigil, gzipSigil})
func NewSigil(name string) (Sigil, error) {
	switch name {
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
		return nil, core.E("sigil.NewSigil", core.Concat("unknown sigil name: ", name), nil)
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

	writeIndent := func(level int) {
		for i := 0; i < level; i++ {
			builder.WriteString("  ")
		}
	}

	for i := 0; i < len(compact); i++ {
		ch := compact[i]
		if inString {
			builder.WriteByte(ch)
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}

		switch ch {
		case '"':
			inString = true
			builder.WriteByte(ch)
		case '{', '[':
			builder.WriteByte(ch)
			if i+1 < len(compact) && compact[i+1] != '}' && compact[i+1] != ']' {
				indent++
				builder.WriteByte('\n')
				writeIndent(indent)
			}
		case '}', ']':
			if i > 0 && compact[i-1] != '{' && compact[i-1] != '[' {
				indent--
				builder.WriteByte('\n')
				writeIndent(indent)
			}
			builder.WriteByte(ch)
		case ',':
			builder.WriteByte(ch)
			builder.WriteByte('\n')
			writeIndent(indent)
		case ':':
			builder.WriteString(": ")
		default:
			builder.WriteByte(ch)
		}
	}

	return builder.String()
}
