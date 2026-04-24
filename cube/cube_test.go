package cube

import (
	"bytes"
	goio "io"
	"testing"

	coreio "dappco.re/go/io"
	"dappco.re/go/io/local"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testKey is a fixed 32-byte key used across cube tests.
var testKey = []byte("0123456789abcdef0123456789abcdef")

func TestCube_New_Good(t *testing.T) {
	inner := coreio.NewMemoryMedium()
	medium, err := New(Options{Inner: inner, Key: testKey})
	require.NoError(t, err)
	require.NotNil(t, medium)
	assert.Same(t, inner, medium.Inner())
}

func TestCube_New_Bad(t *testing.T) {
	// Nil inner medium should return an error.
	_, err := New(Options{Inner: nil, Key: testKey})
	assert.Error(t, err)
}

func TestCube_New_Ugly(t *testing.T) {
	// Wrong key size must be rejected.
	_, err := New(Options{Inner: coreio.NewMemoryMedium(), Key: []byte("short")})
	assert.Error(t, err)
	// Empty key is also invalid.
	_, err = New(Options{Inner: coreio.NewMemoryMedium(), Key: nil})
	assert.Error(t, err)
}

func TestCube_WriteRead_Good(t *testing.T) {
	inner := coreio.NewMemoryMedium()
	medium, err := New(Options{Inner: inner, Key: testKey})
	require.NoError(t, err)

	require.NoError(t, medium.Write("notes/todo.txt", "ship the cube"))

	plaintext, err := medium.Read("notes/todo.txt")
	require.NoError(t, err)
	assert.Equal(t, "ship the cube", plaintext)
}

func TestCube_WriteRead_Bad(t *testing.T) {
	inner := coreio.NewMemoryMedium()
	medium, err := New(Options{Inner: inner, Key: testKey})
	require.NoError(t, err)

	// Read of missing file should return an error.
	_, err = medium.Read("missing.txt")
	assert.Error(t, err)
}

func TestCube_WriteRead_Ugly(t *testing.T) {
	inner := coreio.NewMemoryMedium()
	medium, err := New(Options{Inner: inner, Key: testKey})
	require.NoError(t, err)

	// Underlying storage must contain ciphertext, not plaintext.
	require.NoError(t, medium.Write("secret.txt", "sensitive payload"))
	raw, err := inner.Read("secret.txt")
	require.NoError(t, err)
	assert.NotEqual(t, "sensitive payload", raw, "cube must persist ciphertext, never plaintext")

	// Reading with the wrong key must fail.
	otherKey := []byte("fedcba9876543210fedcba9876543210")
	otherMedium, err := New(Options{Inner: inner, Key: otherKey})
	require.NoError(t, err)
	_, err = otherMedium.Read("secret.txt")
	assert.Error(t, err)
}

func TestCube_WriteMode_Good(t *testing.T) {
	inner := coreio.NewMemoryMedium()
	medium, err := New(Options{Inner: inner, Key: testKey})
	require.NoError(t, err)

	require.NoError(t, medium.WriteMode("keys/private.key", "secret-key", 0600))
	plaintext, err := medium.Read("keys/private.key")
	require.NoError(t, err)
	assert.Equal(t, "secret-key", plaintext)
}

func TestCube_WriteMode_Bad(t *testing.T) {
	inner := coreio.NewMemoryMedium()
	medium, err := New(Options{Inner: inner, Key: testKey})
	require.NoError(t, err)

	// Writing into a path that conflicts with a directory should fail via the inner Medium.
	require.NoError(t, inner.EnsureDir("data"))
	err = medium.WriteMode("data", "payload", 0644)
	assert.Error(t, err)
}

func TestCube_WriteMode_Ugly(t *testing.T) {
	inner := coreio.NewMemoryMedium()
	medium, err := New(Options{Inner: inner, Key: testKey})
	require.NoError(t, err)

	// Empty payload must round-trip.
	require.NoError(t, medium.Write("empty.txt", ""))
	plaintext, err := medium.Read("empty.txt")
	require.NoError(t, err)
	assert.Equal(t, "", plaintext)
}

func TestCube_Streaming_Good(t *testing.T) {
	inner := coreio.NewMemoryMedium()
	medium, err := New(Options{Inner: inner, Key: testKey})
	require.NoError(t, err)

	writer, err := medium.Create("log.txt")
	require.NoError(t, err)
	_, err = writer.Write([]byte("line one\n"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	reader, err := medium.ReadStream("log.txt")
	require.NoError(t, err)
	defer reader.Close()
	content, err := goio.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, "line one\n", string(content))
}

func TestCube_Streaming_Bad(t *testing.T) {
	inner := coreio.NewMemoryMedium()
	medium, err := New(Options{Inner: inner, Key: testKey})
	require.NoError(t, err)

	// Reading a stream that does not exist returns an error.
	_, err = medium.ReadStream("missing.txt")
	assert.Error(t, err)
}

func TestCube_Streaming_Ugly(t *testing.T) {
	inner := coreio.NewMemoryMedium()
	medium, err := New(Options{Inner: inner, Key: testKey})
	require.NoError(t, err)

	// Append must decrypt the existing payload, then append.
	require.NoError(t, medium.Write("log.txt", "line one\n"))
	writer, err := medium.Append("log.txt")
	require.NoError(t, err)
	_, err = writer.Write([]byte("line two\n"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	plaintext, err := medium.Read("log.txt")
	require.NoError(t, err)
	assert.Equal(t, "line one\nline two\n", plaintext)
}

func TestCube_Open_Good(t *testing.T) {
	inner := coreio.NewMemoryMedium()
	medium, err := New(Options{Inner: inner, Key: testKey})
	require.NoError(t, err)
	require.NoError(t, medium.Write("notes.txt", "ship it"))

	file, err := medium.Open("notes.txt")
	require.NoError(t, err)
	defer file.Close()

	buffer := bytes.NewBuffer(nil)
	_, err = goio.Copy(buffer, file)
	require.NoError(t, err)
	assert.Equal(t, "ship it", buffer.String())
}

func TestCube_Open_Bad(t *testing.T) {
	inner := coreio.NewMemoryMedium()
	medium, err := New(Options{Inner: inner, Key: testKey})
	require.NoError(t, err)

	_, err = medium.Open("missing.txt")
	assert.Error(t, err)
}

func TestCube_Open_Ugly(t *testing.T) {
	inner := coreio.NewMemoryMedium()
	medium, err := New(Options{Inner: inner, Key: testKey})
	require.NoError(t, err)

	// Write directly to the inner Medium (plaintext) — cube.Open must fail to decrypt.
	require.NoError(t, inner.Write("secret.txt", "not ciphertext"))
	_, err = medium.Open("secret.txt")
	assert.Error(t, err)
}

func TestCube_PassthroughOperations_Good(t *testing.T) {
	inner := coreio.NewMemoryMedium()
	medium, err := New(Options{Inner: inner, Key: testKey})
	require.NoError(t, err)

	// Exists / IsFile / IsDir / List / Stat pass through to inner.
	require.NoError(t, medium.EnsureDir("data"))
	require.NoError(t, medium.Write("data/one.txt", "alpha"))

	assert.True(t, medium.Exists("data/one.txt"))
	assert.True(t, medium.IsFile("data/one.txt"))
	assert.True(t, medium.IsDir("data"))

	entries, err := medium.List("data")
	require.NoError(t, err)
	assert.NotEmpty(t, entries)

	info, err := medium.Stat("data/one.txt")
	require.NoError(t, err)
	assert.False(t, info.IsDir())
}

func TestCube_PassthroughOperations_Bad(t *testing.T) {
	inner := coreio.NewMemoryMedium()
	medium, err := New(Options{Inner: inner, Key: testKey})
	require.NoError(t, err)

	// Deleting a missing file surfaces the underlying Medium's error.
	err = medium.Delete("missing.txt")
	assert.Error(t, err)
}

func TestCube_PassthroughOperations_Ugly(t *testing.T) {
	inner := coreio.NewMemoryMedium()
	medium, err := New(Options{Inner: inner, Key: testKey})
	require.NoError(t, err)

	// Rename preserves ciphertext semantics.
	require.NoError(t, medium.Write("old.txt", "keep"))
	require.NoError(t, medium.Rename("old.txt", "new.txt"))
	plaintext, err := medium.Read("new.txt")
	require.NoError(t, err)
	assert.Equal(t, "keep", plaintext)

	// DeleteAll removes the entire subtree.
	require.NoError(t, medium.Write("branch/a.txt", "a"))
	require.NoError(t, medium.Write("branch/b.txt", "b"))
	require.NoError(t, medium.DeleteAll("branch"))
	assert.False(t, inner.Exists("branch/a.txt"))
}

func TestCube_Pack_Good(t *testing.T) {
	tempDir := t.TempDir()
	sandbox, err := local.New(tempDir)
	require.NoError(t, err)
	outputPath := tempDir + "/app.cube"

	source := coreio.NewMemoryMedium()
	require.NoError(t, source.Write("config/app.yaml", "port: 8080"))
	require.NoError(t, source.Write("data/user.json", `{"name":"alice"}`))

	require.NoError(t, Pack(outputPath, source, testKey))
	assert.True(t, sandbox.Exists("app.cube"))
}

func TestCube_Pack_Bad(t *testing.T) {
	// Missing source must error.
	err := Pack("output.cube", nil, testKey)
	assert.Error(t, err)

	// Missing output path must error.
	err = Pack("", coreio.NewMemoryMedium(), testKey)
	assert.Error(t, err)
}

func TestCube_Pack_Ugly(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := tempDir + "/bad.cube"

	// Invalid (short) key must error before any filesystem work.
	source := coreio.NewMemoryMedium()
	require.NoError(t, source.Write("a.txt", "payload"))
	err := Pack(outputPath, source, []byte("short"))
	assert.Error(t, err)
}

func TestCube_Unpack_Good(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := tempDir + "/app.cube"

	source := coreio.NewMemoryMedium()
	require.NoError(t, source.Write("config/app.yaml", "port: 8080"))
	require.NoError(t, source.Write("data/user.json", `{"name":"alice"}`))

	require.NoError(t, Pack(outputPath, source, testKey))

	restored := coreio.NewMemoryMedium()
	require.NoError(t, Unpack(outputPath, restored, testKey))

	config, err := restored.Read("config/app.yaml")
	require.NoError(t, err)
	assert.Equal(t, "port: 8080", config)

	user, err := restored.Read("data/user.json")
	require.NoError(t, err)
	assert.Equal(t, `{"name":"alice"}`, user)
}

func TestCube_Unpack_Bad(t *testing.T) {
	err := Unpack("missing.cube", coreio.NewMemoryMedium(), testKey)
	assert.Error(t, err)

	err = Unpack("some.cube", nil, testKey)
	assert.Error(t, err)

	err = Unpack("", coreio.NewMemoryMedium(), testKey)
	assert.Error(t, err)
}

func TestCube_Unpack_Ugly(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := tempDir + "/app.cube"

	source := coreio.NewMemoryMedium()
	require.NoError(t, source.Write("secret.txt", "classified"))
	require.NoError(t, Pack(outputPath, source, testKey))

	// Attempting to unpack with a different key must fail.
	badKey := []byte("fedcba9876543210fedcba9876543210")
	err := Unpack(outputPath, coreio.NewMemoryMedium(), badKey)
	assert.Error(t, err)
}

func TestCube_Open_Packed_Good(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := tempDir + "/app.cube"

	source := coreio.NewMemoryMedium()
	require.NoError(t, source.Write("config/app.yaml", "port: 8080"))
	require.NoError(t, Pack(outputPath, source, testKey))

	cubeMedium, err := Open(outputPath, testKey)
	require.NoError(t, err)

	content, err := cubeMedium.Read("config/app.yaml")
	require.NoError(t, err)
	assert.Equal(t, "port: 8080", content)
}

func TestCube_Open_Packed_Bad(t *testing.T) {
	_, err := Open("", testKey)
	assert.Error(t, err)

	_, err = Open("missing.cube", testKey)
	assert.Error(t, err)
}

func TestCube_Open_Packed_Ugly(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := tempDir + "/app.cube"

	source := coreio.NewMemoryMedium()
	require.NoError(t, source.Write("a.txt", "alpha"))
	require.NoError(t, Pack(outputPath, source, testKey))

	// Wrong key fails.
	badKey := []byte("fedcba9876543210fedcba9876543210")
	_, err := Open(outputPath, badKey)
	assert.Error(t, err)
}

func TestCube_DoubleEncryption_Good(t *testing.T) {
	inner := coreio.NewMemoryMedium()
	userKey := []byte("0123456789abcdef0123456789abcdef")
	transportKey := []byte("fedcba9876543210fedcba9876543210")

	userCube, err := New(Options{Inner: inner, Key: userKey})
	require.NoError(t, err)
	outerCube, err := New(Options{Inner: userCube, Key: transportKey})
	require.NoError(t, err)

	require.NoError(t, outerCube.Write("secret.txt", "classified"))
	plaintext, err := outerCube.Read("secret.txt")
	require.NoError(t, err)
	assert.Equal(t, "classified", plaintext)

	// The underlying inner Medium holds a double-encrypted payload.
	raw, err := inner.Read("secret.txt")
	require.NoError(t, err)
	assert.NotEqual(t, "classified", raw)
}

func TestCube_DoubleEncryption_Bad(t *testing.T) {
	inner := coreio.NewMemoryMedium()
	userKey := []byte("0123456789abcdef0123456789abcdef")
	transportKey := []byte("fedcba9876543210fedcba9876543210")

	userCube, err := New(Options{Inner: inner, Key: userKey})
	require.NoError(t, err)
	outerCube, err := New(Options{Inner: userCube, Key: transportKey})
	require.NoError(t, err)

	require.NoError(t, outerCube.Write("secret.txt", "classified"))

	// Reading through the inner userCube alone returns ciphertext, not plaintext.
	stillEncrypted, err := userCube.Read("secret.txt")
	require.NoError(t, err)
	assert.NotEqual(t, "classified", stillEncrypted)
}

func TestCube_DoubleEncryption_Ugly(t *testing.T) {
	inner := coreio.NewMemoryMedium()
	userKey := []byte("0123456789abcdef0123456789abcdef")
	transportKey := []byte("fedcba9876543210fedcba9876543210")

	userCube, err := New(Options{Inner: inner, Key: userKey})
	require.NoError(t, err)
	outerCube, err := New(Options{Inner: userCube, Key: transportKey})
	require.NoError(t, err)

	require.NoError(t, outerCube.Write("secret.txt", "classified"))

	// Swapping key order must fail to decrypt.
	wrongOrder, err := New(Options{Inner: inner, Key: transportKey})
	require.NoError(t, err)
	_, err = wrongOrder.Read("secret.txt")
	assert.Error(t, err)
}
