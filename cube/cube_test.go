package cube

import (
	"bytes"
	core "dappco.re/go"
	goio "io"

	coreio "dappco.re/go/io"
)

// testKey is a fixed 32-byte key used across cube tests.
var testKey = []byte("0123456789abcdef0123456789abcdef")

func TestCube_New_Good(t *core.T) {
	inner := coreio.NewMemoryMedium()
	medium, err := New(Options{Inner: inner, Key: testKey})
	core.RequireNoError(t, err)
	core.AssertNotNil(t, medium)
	core.AssertSame(t, inner, medium.Inner())
}

func TestCube_New_Bad(t *core.T) {
	medium, err := New(Options{Inner: nil, Key: testKey})
	core.AssertNil(t, medium)
	core.AssertError(t, err)
	if err == nil {
		t.Fatal("expected nil inner medium to fail")
	}
	core.AssertContains(t, err.Error(), "inner medium is required")
}

func TestCube_New_Ugly(t *core.T) {
	// Wrong key size must be rejected.
	_, err := New(Options{Inner: coreio.NewMemoryMedium(), Key: []byte("short")})
	core.AssertError(t, err)
	// Empty key is also invalid.
	_, err = New(Options{Inner: coreio.NewMemoryMedium(), Key: nil})
	core.AssertError(t, err)
}

func TestCube_WriteRead_Good(t *core.T) {
	inner := coreio.NewMemoryMedium()
	medium, err := New(Options{Inner: inner, Key: testKey})
	core.RequireNoError(t, err)

	core.RequireNoError(t, medium.Write("notes/todo.txt", "ship the cube"))

	plaintext, err := medium.Read("notes/todo.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "ship the cube", plaintext)
}

func TestCube_WriteRead_Bad(t *core.T) {
	inner := coreio.NewMemoryMedium()
	medium, err := New(Options{Inner: inner, Key: testKey})
	core.RequireNoError(t, err)

	// Read of missing file should return an error.
	_, err = medium.Read("missing.txt")
	core.AssertError(t, err)
}

func TestCube_WriteRead_Ugly(t *core.T) {
	inner := coreio.NewMemoryMedium()
	medium, err := New(Options{Inner: inner, Key: testKey})
	core.RequireNoError(t, err)

	// Underlying storage must contain ciphertext, not plaintext.
	core.RequireNoError(t, medium.Write("secret.txt", "sensitive payload"))
	raw, err := inner.Read("secret.txt")
	core.RequireNoError(t, err)
	core.AssertNotEqual(t, "sensitive payload", raw, "cube must persist ciphertext, never plaintext")

	// Reading with the wrong key must fail.
	otherKey := []byte("fedcba9876543210fedcba9876543210")
	otherMedium, err := New(Options{Inner: inner, Key: otherKey})
	core.RequireNoError(t, err)
	_, err = otherMedium.Read("secret.txt")
	core.AssertError(t, err)
}

func TestCube_WriteMode_Good(t *core.T) {
	inner := coreio.NewMemoryMedium()
	medium, err := New(Options{Inner: inner, Key: testKey})
	core.RequireNoError(t, err)

	core.RequireNoError(t, medium.WriteMode("keys/private.key", "secret-key", 0600))
	plaintext, err := medium.Read("keys/private.key")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "secret-key", plaintext)
}

func TestCube_WriteMode_Bad(t *core.T) {
	inner := coreio.NewMemoryMedium()
	medium, err := New(Options{Inner: inner, Key: testKey})
	core.RequireNoError(t, err)

	// Writing into a path that conflicts with a directory should fail via the inner Medium.
	core.RequireNoError(t, inner.EnsureDir("data"))
	err = medium.WriteMode("data", "payload", 0644)
	core.AssertError(t, err)
}

func TestCube_WriteMode_Ugly(t *core.T) {
	inner := coreio.NewMemoryMedium()
	medium, err := New(Options{Inner: inner, Key: testKey})
	core.RequireNoError(t, err)

	// Empty payload must round-trip.
	core.RequireNoError(t, medium.Write("empty.txt", ""))
	plaintext, err := medium.Read("empty.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "", plaintext)
}

func TestCube_Streaming_Good(t *core.T) {
	inner := coreio.NewMemoryMedium()
	medium, err := New(Options{Inner: inner, Key: testKey})
	core.RequireNoError(t, err)

	writer, err := medium.Create("log.txt")
	core.RequireNoError(t, err)
	_, err = writer.Write([]byte("line one\n"))
	core.RequireNoError(t, err)
	core.RequireNoError(t, writer.Close())

	reader, err := medium.ReadStream("log.txt")
	core.RequireNoError(t, err)
	defer reader.Close()
	content, err := goio.ReadAll(reader)
	core.RequireNoError(t, err)
	core.AssertEqual(t, "line one\n", string(content))
}

func TestCube_Streaming_Bad(t *core.T) {
	inner := coreio.NewMemoryMedium()
	medium, err := New(Options{Inner: inner, Key: testKey})
	core.RequireNoError(t, err)

	// Reading a stream that does not exist returns an error.
	_, err = medium.ReadStream("missing.txt")
	core.AssertError(t, err)
}

func TestCube_Streaming_Ugly(t *core.T) {
	inner := coreio.NewMemoryMedium()
	medium, err := New(Options{Inner: inner, Key: testKey})
	core.RequireNoError(t, err)

	// Append must decrypt the existing payload, then append.
	core.RequireNoError(t, medium.Write("log.txt", "line one\n"))
	writer, err := medium.Append("log.txt")
	core.RequireNoError(t, err)
	_, err = writer.Write([]byte("line two\n"))
	core.RequireNoError(t, err)
	core.RequireNoError(t, writer.Close())

	plaintext, err := medium.Read("log.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "line one\nline two\n", plaintext)
}

func TestCube_Open_Good(t *core.T) {
	inner := coreio.NewMemoryMedium()
	medium, err := New(Options{Inner: inner, Key: testKey})
	core.RequireNoError(t, err)
	core.RequireNoError(t, medium.Write("notes.txt", "ship it"))

	file, err := medium.Open("notes.txt")
	core.RequireNoError(t, err)
	defer file.Close()

	buffer := bytes.NewBuffer(nil)
	_, err = goio.Copy(buffer, file)
	core.RequireNoError(t, err)
	core.AssertEqual(t, "ship it", buffer.String())
}

func TestCube_Open_Bad(t *core.T) {
	inner := coreio.NewMemoryMedium()
	medium, err := New(Options{Inner: inner, Key: testKey})
	core.RequireNoError(t, err)

	_, err = medium.Open("missing.txt")
	core.AssertError(t, err)
}

func TestCube_Open_Ugly(t *core.T) {
	inner := coreio.NewMemoryMedium()
	medium, err := New(Options{Inner: inner, Key: testKey})
	core.RequireNoError(t, err)

	// Write directly to the inner Medium (plaintext) — cube.Open must fail to decrypt.
	core.RequireNoError(t, inner.Write("secret.txt", "not ciphertext"))
	_, err = medium.Open("secret.txt")
	core.AssertError(t, err)
}

func TestCube_PassthroughOperations_Good(t *core.T) {
	inner := coreio.NewMemoryMedium()
	medium, err := New(Options{Inner: inner, Key: testKey})
	core.RequireNoError(t, err)

	// Exists / IsFile / IsDir / List / Stat pass through to inner.
	core.RequireNoError(t, medium.EnsureDir("data"))
	core.RequireNoError(t, medium.Write("data/one.txt", "alpha"))

	core.AssertTrue(t, medium.Exists("data/one.txt"))
	core.AssertTrue(t, medium.IsFile("data/one.txt"))
	core.AssertTrue(t, medium.IsDir("data"))

	entries, err := medium.List("data")
	core.RequireNoError(t, err)
	core.AssertNotEmpty(t, entries)

	info, err := medium.Stat("data/one.txt")
	core.RequireNoError(t, err)
	core.AssertFalse(t, info.IsDir())
}

func TestCube_PassthroughOperations_Bad(t *core.T) {
	inner := coreio.NewMemoryMedium()
	medium, err := New(Options{Inner: inner, Key: testKey})
	core.RequireNoError(t, err)

	// Deleting a missing file surfaces the underlying Medium's error.
	err = medium.Delete("missing.txt")
	core.AssertError(t, err)
}

func TestCube_PassthroughOperations_Ugly(t *core.T) {
	inner := coreio.NewMemoryMedium()
	medium, err := New(Options{Inner: inner, Key: testKey})
	core.RequireNoError(t, err)

	// Rename preserves ciphertext semantics.
	core.RequireNoError(t, medium.Write("old.txt", "keep"))
	core.RequireNoError(t, medium.Rename("old.txt", "new.txt"))
	plaintext, err := medium.Read("new.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "keep", plaintext)

	// DeleteAll removes the entire subtree.
	core.RequireNoError(t, medium.Write("branch/a.txt", "a"))
	core.RequireNoError(t, medium.Write("branch/b.txt", "b"))
	core.RequireNoError(t, medium.DeleteAll("branch"))
	core.AssertFalse(t, inner.Exists("branch/a.txt"))
}

func TestCube_Pack_Good(t *core.T) {
	tempDir := t.TempDir()
	sandbox, err := coreio.NewSandboxed(tempDir)
	core.RequireNoError(t, err)
	outputPath := tempDir + "/app.cube"

	source := coreio.NewMemoryMedium()
	core.RequireNoError(t, source.Write("config/app.yaml", "port: 8080"))
	core.RequireNoError(t, source.Write("data/user.json", `{"name":"alice"}`))

	core.RequireNoError(t, Pack(outputPath, source, testKey))
	core.AssertTrue(t, sandbox.Exists("app.cube"))
}

func TestCube_Pack_Bad(t *core.T) {
	// Missing source must error.
	err := Pack("output.cube", nil, testKey)
	core.AssertError(t, err)

	// Missing output path must error.
	err = Pack("", coreio.NewMemoryMedium(), testKey)
	core.AssertError(t, err)
}

func TestCube_Pack_Ugly(t *core.T) {
	tempDir := t.TempDir()
	outputPath := tempDir + "/bad.cube"

	// Invalid (short) key must error before any filesystem work.
	source := coreio.NewMemoryMedium()
	core.RequireNoError(t, source.Write("a.txt", "payload"))
	err := Pack(outputPath, source, []byte("short"))
	core.AssertError(t, err)
}

func TestCube_Unpack_Good(t *core.T) {
	tempDir := t.TempDir()
	outputPath := tempDir + "/app.cube"

	source := coreio.NewMemoryMedium()
	core.RequireNoError(t, source.Write("config/app.yaml", "port: 8080"))
	core.RequireNoError(t, source.Write("data/user.json", `{"name":"alice"}`))

	core.RequireNoError(t, Pack(outputPath, source, testKey))

	restored := coreio.NewMemoryMedium()
	core.RequireNoError(t, Unpack(outputPath, restored, testKey))

	config, err := restored.Read("config/app.yaml")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "port: 8080", config)

	user, err := restored.Read("data/user.json")
	core.RequireNoError(t, err)
	core.AssertEqual(t, `{"name":"alice"}`, user)
}

func TestCube_Unpack_Bad(t *core.T) {
	err := Unpack("missing.cube", coreio.NewMemoryMedium(), testKey)
	core.AssertError(t, err)

	err = Unpack("some.cube", nil, testKey)
	core.AssertError(t, err)

	err = Unpack("", coreio.NewMemoryMedium(), testKey)
	core.AssertError(t, err)
}

func TestCube_Unpack_Ugly(t *core.T) {
	tempDir := t.TempDir()
	outputPath := tempDir + "/app.cube"

	source := coreio.NewMemoryMedium()
	core.RequireNoError(t, source.Write("secret.txt", "classified"))
	core.RequireNoError(t, Pack(outputPath, source, testKey))

	// Attempting to unpack with a different key must fail.
	badKey := []byte("fedcba9876543210fedcba9876543210")
	err := Unpack(outputPath, coreio.NewMemoryMedium(), badKey)
	core.AssertError(t, err)
}

func TestCube_Open_Packed_Good(t *core.T) {
	tempDir := t.TempDir()
	outputPath := tempDir + "/app.cube"

	source := coreio.NewMemoryMedium()
	core.RequireNoError(t, source.Write("config/app.yaml", "port: 8080"))
	core.RequireNoError(t, Pack(outputPath, source, testKey))

	cubeMedium, err := Open(outputPath, testKey)
	core.RequireNoError(t, err)

	content, err := cubeMedium.Read("config/app.yaml")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "port: 8080", content)
}

func TestCube_Open_Packed_Bad(t *core.T) {
	_, err := Open("", testKey)
	core.AssertError(t, err)

	_, err = Open("missing.cube", testKey)
	core.AssertError(t, err)
}

func TestCube_Open_Packed_Ugly(t *core.T) {
	tempDir := t.TempDir()
	outputPath := tempDir + "/app.cube"

	source := coreio.NewMemoryMedium()
	core.RequireNoError(t, source.Write("a.txt", "alpha"))
	core.RequireNoError(t, Pack(outputPath, source, testKey))

	// Wrong key fails.
	badKey := []byte("fedcba9876543210fedcba9876543210")
	_, err := Open(outputPath, badKey)
	core.AssertError(t, err)
}

func TestCube_DoubleEncryption_Good(t *core.T) {
	inner := coreio.NewMemoryMedium()
	userKey := []byte("0123456789abcdef0123456789abcdef")
	transportKey := []byte("fedcba9876543210fedcba9876543210")

	userCube, err := New(Options{Inner: inner, Key: userKey})
	core.RequireNoError(t, err)
	outerCube, err := New(Options{Inner: userCube, Key: transportKey})
	core.RequireNoError(t, err)

	core.RequireNoError(t, outerCube.Write("secret.txt", "classified"))
	plaintext, err := outerCube.Read("secret.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "classified", plaintext)

	// The underlying inner Medium holds a double-encrypted payload.
	raw, err := inner.Read("secret.txt")
	core.RequireNoError(t, err)
	core.AssertNotEqual(t, "classified", raw)
}

func TestCube_DoubleEncryption_Bad(t *core.T) {
	inner := coreio.NewMemoryMedium()
	userKey := []byte("0123456789abcdef0123456789abcdef")
	transportKey := []byte("fedcba9876543210fedcba9876543210")

	userCube, err := New(Options{Inner: inner, Key: userKey})
	core.RequireNoError(t, err)
	outerCube, err := New(Options{Inner: userCube, Key: transportKey})
	core.RequireNoError(t, err)

	core.RequireNoError(t, outerCube.Write("secret.txt", "classified"))

	// Reading through the inner userCube alone returns ciphertext, not plaintext.
	stillEncrypted, err := userCube.Read("secret.txt")
	core.RequireNoError(t, err)
	core.AssertNotEqual(t, "classified", stillEncrypted)
}

func TestCube_DoubleEncryption_Ugly(t *core.T) {
	inner := coreio.NewMemoryMedium()
	userKey := []byte("0123456789abcdef0123456789abcdef")
	transportKey := []byte("fedcba9876543210fedcba9876543210")

	userCube, err := New(Options{Inner: inner, Key: userKey})
	core.RequireNoError(t, err)
	outerCube, err := New(Options{Inner: userCube, Key: transportKey})
	core.RequireNoError(t, err)

	core.RequireNoError(t, outerCube.Write("secret.txt", "classified"))

	// Swapping key order must fail to decrypt.
	wrongOrder, err := New(Options{Inner: inner, Key: transportKey})
	core.RequireNoError(t, err)
	_, err = wrongOrder.Read("secret.txt")
	core.AssertError(t, err)
}
