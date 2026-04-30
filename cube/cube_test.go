package cube

import (
	core "dappco.re/go"
	coreio "dappco.re/go/io"
	goio "io"
	"io/fs"
	"time"
)

// testKey is a fixed 32-byte key used across cube tests.
var testKey = []byte("0123456789abcdef0123456789abcdef")

const (
	cubeMissingPath     = "missing.txt"
	cubeSecretPath      = "secret.txt"
	cubeLogPath         = "log.txt"
	cubeLineOneContent  = "line one\n"
	cubeDataOnePath     = "data/one.txt"
	cubeAppOutputSuffix = "/app.cube"
	cubeConfigPath      = "config/app.yaml"
	cubeConfigContent   = "port: 8080"
	cubeUserPath        = "data/user.json"
)

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

func TestCube_WriteReadGood(t *core.T) {
	inner := coreio.NewMemoryMedium()
	medium, err := New(Options{Inner: inner, Key: testKey})
	core.RequireNoError(t, err)

	core.RequireNoError(t, medium.Write("notes/todo.txt", "ship the cube"))

	plaintext, err := medium.Read("notes/todo.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "ship the cube", plaintext)
}

func TestCube_WriteReadBad(t *core.T) {
	inner := coreio.NewMemoryMedium()
	medium, err := New(Options{Inner: inner, Key: testKey})
	core.RequireNoError(t, err)

	// Read of missing file should return an error.
	_, err = medium.Read(cubeMissingPath)
	core.AssertError(t, err)
}

func TestCube_WriteReadUgly(t *core.T) {
	inner := coreio.NewMemoryMedium()
	medium, err := New(Options{Inner: inner, Key: testKey})
	core.RequireNoError(t, err)

	// Underlying storage must contain ciphertext, not plaintext.
	core.RequireNoError(t, medium.Write(cubeSecretPath, "sensitive payload"))
	raw, err := inner.Read(cubeSecretPath)
	core.RequireNoError(t, err)
	core.AssertNotEqual(t, "sensitive payload", raw, "cube must persist ciphertext, never plaintext")

	// Reading with the wrong key must fail.
	otherKey := []byte("fedcba9876543210fedcba9876543210")
	otherMedium, err := New(Options{Inner: inner, Key: otherKey})
	core.RequireNoError(t, err)
	_, err = otherMedium.Read(cubeSecretPath)
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

func TestCube_WriteModeUgly(t *core.T) {
	inner := coreio.NewMemoryMedium()
	medium, err := New(Options{Inner: inner, Key: testKey})
	core.RequireNoError(t, err)

	// Empty payload must round-trip.
	core.RequireNoError(t, medium.Write("empty.txt", ""))
	plaintext, err := medium.Read("empty.txt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "", plaintext)
}

func TestCube_StreamingGood(t *core.T) {
	inner := coreio.NewMemoryMedium()
	medium, err := New(Options{Inner: inner, Key: testKey})
	core.RequireNoError(t, err)

	writer, err := medium.Create(cubeLogPath)
	core.RequireNoError(t, err)
	_, err = writer.Write([]byte(cubeLineOneContent))
	core.RequireNoError(t, err)
	core.RequireNoError(t, writer.Close())

	reader, err := medium.ReadStream(cubeLogPath)
	core.RequireNoError(t, err)
	defer func() { _ = reader.Close() }()
	content, err := goio.ReadAll(reader)
	core.RequireNoError(t, err)
	core.AssertEqual(t, cubeLineOneContent, string(content))
}

func TestCube_StreamingBad(t *core.T) {
	inner := coreio.NewMemoryMedium()
	medium, err := New(Options{Inner: inner, Key: testKey})
	core.RequireNoError(t, err)

	// Reading a stream that does not exist returns an error.
	_, err = medium.ReadStream(cubeMissingPath)
	core.AssertError(t, err)
}

func TestCube_StreamingUgly(t *core.T) {
	inner := coreio.NewMemoryMedium()
	medium, err := New(Options{Inner: inner, Key: testKey})
	core.RequireNoError(t, err)

	// Append must decrypt the existing payload, then append.
	core.RequireNoError(t, medium.Write(cubeLogPath, cubeLineOneContent))
	writer, err := medium.Append(cubeLogPath)
	core.RequireNoError(t, err)
	_, err = writer.Write([]byte("line two\n"))
	core.RequireNoError(t, err)
	core.RequireNoError(t, writer.Close())

	plaintext, err := medium.Read(cubeLogPath)
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
	defer func() { _ = file.Close() }()

	buffer := core.NewBuffer()
	_, err = goio.Copy(buffer, file)
	core.RequireNoError(t, err)
	core.AssertEqual(t, "ship it", buffer.String())
}

func TestCube_Open_Bad(t *core.T) {
	inner := coreio.NewMemoryMedium()
	medium, err := New(Options{Inner: inner, Key: testKey})
	core.RequireNoError(t, err)

	_, err = medium.Open(cubeMissingPath)
	core.AssertError(t, err)
}

func TestCube_Open_Ugly(t *core.T) {
	inner := coreio.NewMemoryMedium()
	medium, err := New(Options{Inner: inner, Key: testKey})
	core.RequireNoError(t, err)

	// Write directly to the inner Medium (plaintext) — cube.Open must fail to decrypt.
	core.RequireNoError(t, inner.Write(cubeSecretPath, "not ciphertext"))
	_, err = medium.Open(cubeSecretPath)
	core.AssertError(t, err)
}

func TestCube_PassthroughOperationsGood(t *core.T) {
	inner := coreio.NewMemoryMedium()
	medium, err := New(Options{Inner: inner, Key: testKey})
	core.RequireNoError(t, err)

	// Exists / IsFile / IsDir / List / Stat pass through to inner.
	core.RequireNoError(t, medium.EnsureDir("data"))
	core.RequireNoError(t, medium.Write(cubeDataOnePath, "alpha"))

	core.AssertTrue(t, medium.Exists(cubeDataOnePath))
	core.AssertTrue(t, medium.IsFile(cubeDataOnePath))
	core.AssertTrue(t, medium.IsDir("data"))

	entries, err := medium.List("data")
	core.RequireNoError(t, err)
	core.AssertNotEmpty(t, entries)

	info, err := medium.Stat(cubeDataOnePath)
	core.RequireNoError(t, err)
	core.AssertFalse(t, info.IsDir())
}

func TestCube_PassthroughOperationsBad(t *core.T) {
	inner := coreio.NewMemoryMedium()
	medium, err := New(Options{Inner: inner, Key: testKey})
	core.RequireNoError(t, err)

	// Deleting a missing file surfaces the underlying Medium's error.
	err = medium.Delete(cubeMissingPath)
	core.AssertError(t, err)
}

func TestCube_PassthroughOperationsUgly(t *core.T) {
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
	outputPath := tempDir + cubeAppOutputSuffix

	source := coreio.NewMemoryMedium()
	core.RequireNoError(t, source.Write(cubeConfigPath, cubeConfigContent))
	core.RequireNoError(t, source.Write(cubeUserPath, `{"name":"alice"}`))

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
	outputPath := tempDir + cubeAppOutputSuffix

	source := coreio.NewMemoryMedium()
	core.RequireNoError(t, source.Write(cubeConfigPath, cubeConfigContent))
	core.RequireNoError(t, source.Write(cubeUserPath, `{"name":"alice"}`))

	core.RequireNoError(t, Pack(outputPath, source, testKey))

	restored := coreio.NewMemoryMedium()
	core.RequireNoError(t, Unpack(outputPath, restored, testKey))

	config, err := restored.Read(cubeConfigPath)
	core.RequireNoError(t, err)
	core.AssertEqual(t, cubeConfigContent, config)

	user, err := restored.Read(cubeUserPath)
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
	outputPath := tempDir + cubeAppOutputSuffix

	source := coreio.NewMemoryMedium()
	core.RequireNoError(t, source.Write(cubeSecretPath, "classified"))
	core.RequireNoError(t, Pack(outputPath, source, testKey))

	// Attempting to unpack with a different key must fail.
	badKey := []byte("fedcba9876543210fedcba9876543210")
	err := Unpack(outputPath, coreio.NewMemoryMedium(), badKey)
	core.AssertError(t, err)
}

func TestCube_Open_Packed_Good(t *core.T) {
	tempDir := t.TempDir()
	outputPath := tempDir + cubeAppOutputSuffix

	source := coreio.NewMemoryMedium()
	core.RequireNoError(t, source.Write(cubeConfigPath, cubeConfigContent))
	core.RequireNoError(t, Pack(outputPath, source, testKey))

	cubeMedium, err := Open(outputPath, testKey)
	core.RequireNoError(t, err)

	content, err := cubeMedium.Read(cubeConfigPath)
	core.RequireNoError(t, err)
	core.AssertEqual(t, cubeConfigContent, content)
}

func TestCube_Open_Packed_Bad(t *core.T) {
	_, err := Open("", testKey)
	core.AssertError(t, err)

	_, err = Open("missing.cube", testKey)
	core.AssertError(t, err)
}

func TestCube_Open_Packed_Ugly(t *core.T) {
	tempDir := t.TempDir()
	outputPath := tempDir + cubeAppOutputSuffix

	source := coreio.NewMemoryMedium()
	core.RequireNoError(t, source.Write("a.txt", "alpha"))
	core.RequireNoError(t, Pack(outputPath, source, testKey))

	// Wrong key fails.
	badKey := []byte("fedcba9876543210fedcba9876543210")
	_, err := Open(outputPath, badKey)
	core.AssertError(t, err)
}

func TestCube_DoubleEncryptionGood(t *core.T) {
	inner := coreio.NewMemoryMedium()
	userKey := []byte("0123456789abcdef0123456789abcdef")
	transportKey := []byte("fedcba9876543210fedcba9876543210")

	userCube, err := New(Options{Inner: inner, Key: userKey})
	core.RequireNoError(t, err)
	outerCube, err := New(Options{Inner: userCube, Key: transportKey})
	core.RequireNoError(t, err)

	core.RequireNoError(t, outerCube.Write(cubeSecretPath, "classified"))
	plaintext, err := outerCube.Read(cubeSecretPath)
	core.RequireNoError(t, err)
	core.AssertEqual(t, "classified", plaintext)

	// The underlying inner Medium holds a double-encrypted payload.
	raw, err := inner.Read(cubeSecretPath)
	core.RequireNoError(t, err)
	core.AssertNotEqual(t, "classified", raw)
}

func TestCube_DoubleEncryptionBad(t *core.T) {
	inner := coreio.NewMemoryMedium()
	userKey := []byte("0123456789abcdef0123456789abcdef")
	transportKey := []byte("fedcba9876543210fedcba9876543210")

	userCube, err := New(Options{Inner: inner, Key: userKey})
	core.RequireNoError(t, err)
	outerCube, err := New(Options{Inner: userCube, Key: transportKey})
	core.RequireNoError(t, err)

	core.RequireNoError(t, outerCube.Write(cubeSecretPath, "classified"))

	// Reading through the inner userCube alone returns ciphertext, not plaintext.
	stillEncrypted, err := userCube.Read(cubeSecretPath)
	core.RequireNoError(t, err)
	core.AssertNotEqual(t, "classified", stillEncrypted)
}

func TestCube_DoubleEncryptionUgly(t *core.T) {
	inner := coreio.NewMemoryMedium()
	userKey := []byte("0123456789abcdef0123456789abcdef")
	transportKey := []byte("fedcba9876543210fedcba9876543210")

	userCube, err := New(Options{Inner: inner, Key: userKey})
	core.RequireNoError(t, err)
	outerCube, err := New(Options{Inner: userCube, Key: transportKey})
	core.RequireNoError(t, err)

	core.RequireNoError(t, outerCube.Write(cubeSecretPath, "classified"))

	// Swapping key order must fail to decrypt.
	wrongOrder, err := New(Options{Inner: inner, Key: transportKey})
	core.RequireNoError(t, err)
	_, err = wrongOrder.Read(cubeSecretPath)
	core.AssertError(t, err)
}

func newCubeMediumFixture(t *core.T) (*Medium, coreio.Medium) {
	t.Helper()

	inner := coreio.NewMemoryMedium()
	medium, err := New(Options{Inner: inner, Key: testKey})
	core.RequireNoError(t, err)
	return medium, inner
}

func TestCube_Medium_Inner_Good(t *core.T) {
	medium, inner := newCubeMediumFixture(t)
	got := medium.Inner()
	core.AssertSame(t, inner, got)
}

func TestCube_Medium_Inner_Bad(t *core.T) {
	medium := &Medium{}
	got := medium.Inner()
	core.AssertNil(t, got)
}

func TestCube_Medium_Inner_Ugly(t *core.T) {
	medium, inner := newCubeMediumFixture(t)
	core.RequireNoError(t, inner.Write("raw.txt", "ciphertext"))
	got := medium.Inner()
	core.AssertTrue(t, got.Exists("raw.txt"))
}

func TestCube_Medium_Read_Good(t *core.T) {
	medium, _ := newCubeMediumFixture(t)
	core.RequireNoError(t, medium.Write("read.txt", "payload"))
	got, err := medium.Read("read.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "payload", got)
}

func TestCube_Medium_Read_Bad(t *core.T) {
	medium, _ := newCubeMediumFixture(t)
	got, err := medium.Read(cubeMissingPath)
	core.AssertError(t, err)
	core.AssertEqual(t, "", got)
}

func TestCube_Medium_Read_Ugly(t *core.T) {
	medium, inner := newCubeMediumFixture(t)
	core.RequireNoError(t, inner.Write("raw.txt", "not ciphertext"))
	got, err := medium.Read("raw.txt")
	core.AssertError(t, err)
	core.AssertEqual(t, "", got)
}

func TestCube_Medium_Write_Good(t *core.T) {
	medium, inner := newCubeMediumFixture(t)
	err := medium.Write("write.txt", "payload")
	core.AssertNoError(t, err)
	core.AssertTrue(t, inner.IsFile("write.txt"))
}

func TestCube_Medium_Write_Bad(t *core.T) {
	medium, inner := newCubeMediumFixture(t)
	core.RequireNoError(t, inner.EnsureDir("dir"))
	err := medium.Write("dir", "payload")
	core.AssertError(t, err)
}

func TestCube_Medium_Write_Ugly(t *core.T) {
	medium, _ := newCubeMediumFixture(t)
	err := medium.Write("empty.txt", "")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.Exists("empty.txt"))
}

func TestCube_Medium_WriteMode_Good(t *core.T) {
	medium, _ := newCubeMediumFixture(t)
	err := medium.WriteMode("mode.txt", "payload", 0600)
	info, statErr := medium.Stat("mode.txt")
	core.AssertNoError(t, err)
	core.AssertNoError(t, statErr)
	core.AssertEqual(t, fs.FileMode(0600), info.Mode().Perm())
}

func TestCube_Medium_WriteMode_Bad(t *core.T) {
	medium, inner := newCubeMediumFixture(t)
	core.RequireNoError(t, inner.EnsureDir("dir"))
	err := medium.WriteMode("dir", "payload", 0600)
	core.AssertError(t, err)
}

func TestCube_Medium_WriteMode_Ugly(t *core.T) {
	medium, _ := newCubeMediumFixture(t)
	err := medium.WriteMode("zero-mode.txt", "payload", 0)
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.Exists("zero-mode.txt"))
}

func TestCube_Medium_EnsureDir_Good(t *core.T) {
	medium, _ := newCubeMediumFixture(t)
	err := medium.EnsureDir("dir")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsDir("dir"))
}

func TestCube_Medium_EnsureDir_Bad(t *core.T) {
	medium, _ := newCubeMediumFixture(t)
	core.RequireNoError(t, medium.Write("file.txt", "payload"))
	err := medium.EnsureDir("file.txt")
	core.AssertError(t, err)
}

func TestCube_Medium_EnsureDir_Ugly(t *core.T) {
	medium, _ := newCubeMediumFixture(t)
	err := medium.EnsureDir("a/b/c")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsDir("a/b/c"))
}

func TestCube_Medium_IsFile_Good(t *core.T) {
	medium, _ := newCubeMediumFixture(t)
	core.RequireNoError(t, medium.Write("file.txt", "payload"))
	got := medium.IsFile("file.txt")
	core.AssertTrue(t, got)
}

func TestCube_Medium_IsFile_Bad(t *core.T) {
	medium, _ := newCubeMediumFixture(t)
	got := medium.IsFile(cubeMissingPath)
	core.AssertFalse(t, got)
}

func TestCube_Medium_IsFile_Ugly(t *core.T) {
	medium, _ := newCubeMediumFixture(t)
	core.RequireNoError(t, medium.EnsureDir("dir"))
	got := medium.IsFile("dir")
	core.AssertFalse(t, got)
}

func TestCube_Medium_Delete_Good(t *core.T) {
	medium, _ := newCubeMediumFixture(t)
	core.RequireNoError(t, medium.Write("delete.txt", "payload"))
	err := medium.Delete("delete.txt")
	core.AssertNoError(t, err)
	core.AssertFalse(t, medium.Exists("delete.txt"))
}

func TestCube_Medium_Delete_Bad(t *core.T) {
	medium, _ := newCubeMediumFixture(t)
	err := medium.Delete(cubeMissingPath)
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists(cubeMissingPath))
}

func TestCube_Medium_Delete_Ugly(t *core.T) {
	medium, _ := newCubeMediumFixture(t)
	core.RequireNoError(t, medium.EnsureDir("empty"))
	err := medium.Delete("empty")
	core.AssertNoError(t, err)
	core.AssertFalse(t, medium.Exists("empty"))
}

func TestCube_Medium_DeleteAll_Good(t *core.T) {
	medium, _ := newCubeMediumFixture(t)
	core.RequireNoError(t, medium.Write("tree/file.txt", "payload"))
	err := medium.DeleteAll("tree")
	core.AssertNoError(t, err)
	core.AssertFalse(t, medium.Exists("tree/file.txt"))
}

func TestCube_Medium_DeleteAll_Bad(t *core.T) {
	medium, _ := newCubeMediumFixture(t)
	err := medium.DeleteAll("missing")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists("missing"))
}

func TestCube_Medium_DeleteAll_Ugly(t *core.T) {
	medium, _ := newCubeMediumFixture(t)
	err := medium.DeleteAll("")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists("anything"))
}

func TestCube_Medium_Rename_Good(t *core.T) {
	medium, _ := newCubeMediumFixture(t)
	core.RequireNoError(t, medium.Write("old.txt", "payload"))
	err := medium.Rename("old.txt", "new.txt")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile("new.txt"))
}

func TestCube_Medium_Rename_Bad(t *core.T) {
	medium, _ := newCubeMediumFixture(t)
	err := medium.Rename(cubeMissingPath, "new.txt")
	core.AssertError(t, err)
	core.AssertFalse(t, medium.Exists("new.txt"))
}

func TestCube_Medium_Rename_Ugly(t *core.T) {
	medium, _ := newCubeMediumFixture(t)
	core.RequireNoError(t, medium.Write("dir/old.txt", "payload"))
	err := medium.Rename("dir", "moved")
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.IsFile("moved/old.txt"))
}

func TestCube_Medium_List_Good(t *core.T) {
	medium, _ := newCubeMediumFixture(t)
	core.RequireNoError(t, medium.Write("dir/a.txt", "payload"))
	entries, err := medium.List("dir")
	core.AssertNoError(t, err)
	core.AssertLen(t, entries, 1)
}

func TestCube_Medium_List_Bad(t *core.T) {
	medium, _ := newCubeMediumFixture(t)
	entries, err := medium.List("missing")
	core.AssertError(t, err)
	core.AssertNil(t, entries)
}

func TestCube_Medium_List_Ugly(t *core.T) {
	medium, _ := newCubeMediumFixture(t)
	entries, err := medium.List("")
	core.AssertNoError(t, err)
	core.AssertEmpty(t, entries)
}

func TestCube_Medium_Stat_Good(t *core.T) {
	medium, _ := newCubeMediumFixture(t)
	core.RequireNoError(t, medium.Write("stat.txt", "payload"))
	info, err := medium.Stat("stat.txt")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "stat.txt", info.Name())
}

func TestCube_Medium_Stat_Bad(t *core.T) {
	medium, _ := newCubeMediumFixture(t)
	info, err := medium.Stat(cubeMissingPath)
	core.AssertError(t, err)
	core.AssertNil(t, info)
}

func TestCube_Medium_Stat_Ugly(t *core.T) {
	medium, _ := newCubeMediumFixture(t)
	core.RequireNoError(t, medium.EnsureDir("dir"))
	info, err := medium.Stat("dir")
	core.AssertNoError(t, err)
	core.AssertTrue(t, info.IsDir())
}

func TestCube_Medium_Open_Good(t *core.T) {
	medium, _ := newCubeMediumFixture(t)
	core.RequireNoError(t, medium.Write("open.txt", "payload"))
	file, err := medium.Open("open.txt")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, file)
	core.RequireNoError(t, file.Close())
}

func TestCube_Medium_Open_Bad(t *core.T) {
	medium, _ := newCubeMediumFixture(t)
	file, err := medium.Open(cubeMissingPath)
	core.AssertError(t, err)
	core.AssertNil(t, file)
}

func TestCube_Medium_Open_Ugly(t *core.T) {
	medium, inner := newCubeMediumFixture(t)
	core.RequireNoError(t, inner.Write("raw.txt", "not ciphertext"))
	file, err := medium.Open("raw.txt")
	core.AssertError(t, err)
	core.AssertNil(t, file)
}

func TestCube_Medium_Create_Good(t *core.T) {
	medium, _ := newCubeMediumFixture(t)
	writer, err := medium.Create("create.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("payload"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestCube_Medium_Create_Bad(t *core.T) {
	medium, _ := newCubeMediumFixture(t)
	writer, err := medium.Create("")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, writer)
}

func TestCube_Medium_Create_Ugly(t *core.T) {
	medium, _ := newCubeMediumFixture(t)
	writer, err := medium.Create("nested/create.txt")
	core.RequireNoError(t, err)
	core.AssertNoError(t, writer.Close())
	core.AssertTrue(t, medium.Exists("nested/create.txt"))
}

func TestCube_Medium_Append_Good(t *core.T) {
	medium, _ := newCubeMediumFixture(t)
	core.RequireNoError(t, medium.Write("append.txt", "a"))
	writer, err := medium.Append("append.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("b"))
	core.RequireNoError(t, writeErr)
	core.RequireNoError(t, writer.Close())
}

func TestCube_Medium_Append_Bad(t *core.T) {
	medium, inner := newCubeMediumFixture(t)
	core.RequireNoError(t, inner.Write("raw.txt", "not ciphertext"))
	writer, err := medium.Append("raw.txt")
	core.AssertError(t, err)
	core.AssertNil(t, writer)
}

func TestCube_Medium_Append_Ugly(t *core.T) {
	medium, _ := newCubeMediumFixture(t)
	writer, err := medium.Append("new.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("created"))
	core.RequireNoError(t, writeErr)
	core.RequireNoError(t, writer.Close())
}

func TestCube_Medium_ReadStream_Good(t *core.T) {
	medium, _ := newCubeMediumFixture(t)
	core.RequireNoError(t, medium.Write("stream.txt", "payload"))
	reader, err := medium.ReadStream("stream.txt")
	core.RequireNoError(t, err)
	defer func() { _ = reader.Close() }()
	data, readErr := goio.ReadAll(reader)
	core.AssertNoError(t, readErr)
	core.AssertEqual(t, "payload", string(data))
}

func TestCube_Medium_ReadStream_Bad(t *core.T) {
	medium, _ := newCubeMediumFixture(t)
	reader, err := medium.ReadStream(cubeMissingPath)
	core.AssertError(t, err)
	core.AssertNil(t, reader)
}

func TestCube_Medium_ReadStream_Ugly(t *core.T) {
	medium, _ := newCubeMediumFixture(t)
	core.RequireNoError(t, medium.Write("empty.txt", ""))
	reader, err := medium.ReadStream("empty.txt")
	core.RequireNoError(t, err)
	core.AssertNoError(t, reader.Close())
}

func TestCube_Medium_WriteStream_Good(t *core.T) {
	medium, _ := newCubeMediumFixture(t)
	writer, err := medium.WriteStream("stream-write.txt")
	core.RequireNoError(t, err)
	_, writeErr := writer.Write([]byte("payload"))
	core.AssertNoError(t, writeErr)
	core.AssertNoError(t, writer.Close())
}

func TestCube_Medium_WriteStream_Bad(t *core.T) {
	medium, _ := newCubeMediumFixture(t)
	writer, err := medium.WriteStream("")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, writer)
}

func TestCube_Medium_WriteStream_Ugly(t *core.T) {
	medium, _ := newCubeMediumFixture(t)
	writer, err := medium.WriteStream("nested/stream.txt")
	core.RequireNoError(t, err)
	core.AssertNoError(t, writer.Close())
	core.AssertTrue(t, medium.Exists("nested/stream.txt"))
}

func TestCube_Medium_Exists_Good(t *core.T) {
	medium, _ := newCubeMediumFixture(t)
	core.RequireNoError(t, medium.Write("exists.txt", "payload"))
	got := medium.Exists("exists.txt")
	core.AssertTrue(t, got)
}

func TestCube_Medium_Exists_Bad(t *core.T) {
	medium, _ := newCubeMediumFixture(t)
	got := medium.Exists(cubeMissingPath)
	core.AssertFalse(t, got)
}

func TestCube_Medium_Exists_Ugly(t *core.T) {
	medium, _ := newCubeMediumFixture(t)
	core.RequireNoError(t, medium.EnsureDir("dir"))
	got := medium.Exists("dir")
	core.AssertTrue(t, got)
}

func TestCube_Medium_IsDir_Good(t *core.T) {
	medium, _ := newCubeMediumFixture(t)
	core.RequireNoError(t, medium.EnsureDir("dir"))
	got := medium.IsDir("dir")
	core.AssertTrue(t, got)
}

func TestCube_Medium_IsDir_Bad(t *core.T) {
	medium, _ := newCubeMediumFixture(t)
	got := medium.IsDir("missing")
	core.AssertFalse(t, got)
}

func TestCube_Medium_IsDir_Ugly(t *core.T) {
	medium, _ := newCubeMediumFixture(t)
	core.RequireNoError(t, medium.Write("file.txt", "payload"))
	got := medium.IsDir("file.txt")
	core.AssertFalse(t, got)
}

func TestCube_File_Stat_Good(t *core.T) {
	file := &cubeFile{name: "file.txt", content: []byte("payload"), mode: 0600, modTime: time.Unix(1, 0)}
	info, err := file.Stat()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "file.txt", info.Name())
}

func TestCube_File_Stat_Bad(t *core.T) {
	file := &cubeFile{}
	info, err := file.Stat()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "", info.Name())
}

func TestCube_File_Stat_Ugly(t *core.T) {
	file := &cubeFile{name: "empty.txt", content: nil}
	info, err := file.Stat()
	core.AssertNoError(t, err)
	core.AssertEqual(t, int64(0), info.Size())
}

func TestCube_File_Read_Good(t *core.T) {
	file := &cubeFile{content: []byte("payload")}
	buffer := make([]byte, 7)
	count, err := file.Read(buffer)
	core.AssertNoError(t, err)
	core.AssertEqual(t, 7, count)
}

func TestCube_File_Read_Bad(t *core.T) {
	file := &cubeFile{content: []byte("x"), offset: 1}
	buffer := make([]byte, 1)
	count, err := file.Read(buffer)
	core.AssertErrorIs(t, err, goio.EOF)
	core.AssertEqual(t, 0, count)
}

func TestCube_File_Read_Ugly(t *core.T) {
	file := &cubeFile{content: []byte("payload")}
	buffer := make([]byte, 3)
	count, err := file.Read(buffer)
	core.AssertNoError(t, err)
	core.AssertEqual(t, "pay", string(buffer[:count]))
}

func TestCube_File_Close_Good(t *core.T) {
	file := &cubeFile{name: "file.txt"}
	err := file.Close()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "file.txt", file.name)
}

func TestCube_File_Close_Bad(t *core.T) {
	file := &cubeFile{}
	err := file.Close()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "", file.name)
}

func TestCube_File_Close_Ugly(t *core.T) {
	file := &cubeFile{offset: 99}
	err := file.Close()
	core.AssertNoError(t, err)
	core.AssertEqual(t, int64(99), file.offset)
}

func TestCube_WriteCloser_Write_Good(t *core.T) {
	writer := &cubeWriteCloser{}
	count, err := writer.Write([]byte("payload"))
	core.AssertNoError(t, err)
	core.AssertEqual(t, len("payload"), count)
}

func TestCube_WriteCloser_Write_Bad(t *core.T) {
	writer := &cubeWriteCloser{}
	count, err := writer.Write(nil)
	core.AssertNoError(t, err)
	core.AssertEqual(t, 0, count)
}

func TestCube_WriteCloser_Write_Ugly(t *core.T) {
	writer := &cubeWriteCloser{data: []byte("a")}
	count, err := writer.Write([]byte("b"))
	core.AssertNoError(t, err)
	core.AssertEqual(t, "ab", string(writer.data[:count+1]))
}

func TestCube_WriteCloser_Close_Good(t *core.T) {
	medium, _ := newCubeMediumFixture(t)
	writer := &cubeWriteCloser{medium: medium, path: "close.txt", data: []byte("payload"), mode: 0600}
	err := writer.Close()
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.Exists("close.txt"))
}

func TestCube_WriteCloser_Close_Bad(t *core.T) {
	writer := &cubeWriteCloser{}
	core.AssertPanics(t, func() { _ = writer.Close() })
	core.AssertNil(t, writer.medium)
}

func TestCube_WriteCloser_Close_Ugly(t *core.T) {
	medium, _ := newCubeMediumFixture(t)
	writer := &cubeWriteCloser{medium: medium, path: "default-mode.txt", data: []byte("payload")}
	err := writer.Close()
	core.AssertNoError(t, err)
	core.AssertTrue(t, medium.Exists("default-mode.txt"))
}

func TestCube_ArchiveBuffer_Write_Good(t *core.T) {
	buffer := &cubeArchiveBuffer{}
	count, err := buffer.Write([]byte("payload"))
	core.AssertNoError(t, err)
	core.AssertEqual(t, len("payload"), count)
}

func TestCube_ArchiveBuffer_Write_Bad(t *core.T) {
	buffer := &cubeArchiveBuffer{}
	count, err := buffer.Write(nil)
	core.AssertNoError(t, err)
	core.AssertEqual(t, 0, count)
}

func TestCube_ArchiveBuffer_Write_Ugly(t *core.T) {
	buffer := &cubeArchiveBuffer{data: []byte("a")}
	count, err := buffer.Write([]byte("b"))
	core.AssertNoError(t, err)
	core.AssertEqual(t, 1, count)
}
