// Example: inner := io.NewMemoryMedium()
// Example: medium, _ := cube.New(cube.Options{Inner: inner, Key: key})
// Example: _ = medium.Write("secret.txt", "classified")
// Example: plain, _ := medium.Read("secret.txt")
package cube

import (
	"archive/tar" // AX-6-exception: tar archive transport has no core equivalent.
	goio "io"     // AX-6-exception: io interface types have no core equivalent; io.EOF preserves stream semantics.
	"io/fs"       // AX-6-exception: fs interface types have no core equivalent.
	"path"        // AX-6-exception: tar entry names use slash-separated paths.
	"time"        // AX-6-exception: filesystem metadata timestamps have no core equivalent.

	core "dappco.re/go"
	coreio "dappco.re/go/io"
	"dappco.re/go/io/node"
	"dappco.re/go/io/sigil"
)

const (
	opCubeNew       = "cube.New"
	opCubeOpen      = "cube.Open"
	opCubePack      = "cube.Pack"
	opCubeUnpack    = "cube.Unpack"
	opCubeArchive   = "cube.archive"
	opCubeExtract   = "cube.extract"
	errCreateCipher = "failed to create cipher"
)

// Example: medium, _ := cube.New(cube.Options{Inner: inner, Key: key})
// Example: _ = medium.Write("secret.txt", "classified")
// Example: plain, _ := medium.Read("secret.txt")
type Medium struct {
	inner coreio.Medium
	sigil *sigil.ChaChaPolySigil
}

var _ coreio.Medium = (*Medium)(nil)

// Example: medium, _ := cube.New(cube.Options{Inner: io.NewMemoryMedium(), Key: key})
type Options struct {
	Inner coreio.Medium
	Key   []byte
}

// Example: medium, _ := cube.New(cube.Options{Inner: io.NewMemoryMedium(), Key: key})
// Example: _ = medium.Write("secret.txt", "classified")
// Example: plaintext, _ := medium.Read("secret.txt")
func New(options Options) (*Medium, error) {
	if options.Inner == nil {
		return nil, core.E(opCubeNew, "inner medium is required", fs.ErrInvalid)
	}
	cipherSigil, err := sigil.NewChaChaPolySigil(options.Key, nil)
	if err != nil {
		return nil, core.E(opCubeNew, "failed to create cipher sigil", err)
	}
	return &Medium{
		inner: options.Inner,
		sigil: cipherSigil,
	}, nil
}

// Example: inner := medium.Inner()
func (medium *Medium) Inner() coreio.Medium {
	return medium.inner
}

// Example: content, _ := medium.Read("secret.txt")
func (medium *Medium) Read(path string) (string, error) {
	ciphertext, err := medium.inner.Read(path)
	if err != nil {
		return "", err
	}
	plaintext, err := sigil.Untransmute([]byte(ciphertext), []sigil.Sigil{medium.sigil})
	if err != nil {
		return "", core.E("cube.Read", core.Concat("failed to decrypt: ", path), err)
	}
	return string(plaintext), nil
}

// Example: _ = medium.Write("secret.txt", "classified")
func (medium *Medium) Write(path, content string) error {
	return medium.WriteMode(path, content, 0644)
}

// Example: _ = medium.WriteMode("keys/private.key", key, 0600)
func (medium *Medium) WriteMode(path, content string, mode fs.FileMode) error {
	ciphertext, err := sigil.Transmute([]byte(content), []sigil.Sigil{medium.sigil})
	if err != nil {
		return core.E("cube.WriteMode", core.Concat("failed to encrypt: ", path), err)
	}
	return medium.inner.WriteMode(path, string(ciphertext), mode)
}

// Example: _ = medium.EnsureDir("data")
func (medium *Medium) EnsureDir(path string) error {
	return medium.inner.EnsureDir(path)
}

// Example: isFile := medium.IsFile("secret.txt")
func (medium *Medium) IsFile(path string) bool {
	return medium.inner.IsFile(path)
}

// Example: _ = medium.Delete("secret.txt")
func (medium *Medium) Delete(path string) error {
	return medium.inner.Delete(path)
}

// Example: _ = medium.DeleteAll("archive")
func (medium *Medium) DeleteAll(path string) error {
	return medium.inner.DeleteAll(path)
}

// Example: _ = medium.Rename("draft.txt", "final.txt")
func (medium *Medium) Rename(oldPath, newPath string) error {
	return medium.inner.Rename(oldPath, newPath)
}

// Example: entries, _ := medium.List("data")
func (medium *Medium) List(path string) ([]fs.DirEntry, error) {
	return medium.inner.List(path)
}

// Example: info, _ := medium.Stat("secret.txt")
func (medium *Medium) Stat(path string) (fs.FileInfo, error) {
	return medium.inner.Stat(path)
}

// Example: file, _ := medium.Open("secret.txt")
func (medium *Medium) Open(path string) (fs.File, error) {
	// Read via cube semantics (decrypt) then wrap in an in-memory fs.File.
	ciphertext, err := medium.inner.Read(path)
	if err != nil {
		return nil, err
	}
	plaintext, err := sigil.Untransmute([]byte(ciphertext), []sigil.Sigil{medium.sigil})
	if err != nil {
		return nil, core.E(opCubeOpen, core.Concat("failed to decrypt: ", path), err)
	}
	info, err := medium.inner.Stat(path)
	if err != nil {
		info = coreio.NewFileInfo(core.PathBase(path), int64(len(plaintext)), 0644, time.Now(), false)
	}
	return &cubeFile{
		name:    core.PathBase(path),
		content: plaintext,
		mode:    info.Mode(),
		modTime: info.ModTime(),
	}, nil
}

// Example: writer, _ := medium.Create("secret.txt")
func (medium *Medium) Create(path string) (goio.WriteCloser, error) {
	return &cubeWriteCloser{medium: medium, path: path, mode: 0644}, nil
}

// Example: writer, _ := medium.Append("log.txt")
func (medium *Medium) Append(path string) (goio.WriteCloser, error) {
	var existing []byte
	if medium.inner.Exists(path) {
		plain, err := medium.Read(path)
		if err != nil {
			return nil, err
		}
		existing = []byte(plain)
	}
	return &cubeWriteCloser{medium: medium, path: path, data: existing, mode: 0644}, nil
}

// Example: reader, _ := medium.ReadStream("secret.txt")
func (medium *Medium) ReadStream(path string) (goio.ReadCloser, error) {
	file, err := medium.Open(path)
	if err != nil {
		return nil, err
	}
	return file, nil
}

// Example: writer, _ := medium.WriteStream("secret.txt")
func (medium *Medium) WriteStream(path string) (goio.WriteCloser, error) {
	return medium.Create(path)
}

// Example: exists := medium.Exists("secret.txt")
func (medium *Medium) Exists(path string) bool {
	return medium.inner.Exists(path)
}

// Example: isDirectory := medium.IsDir("data")
func (medium *Medium) IsDir(path string) bool {
	return medium.inner.IsDir(path)
}

// cubeFile implements fs.File over decrypted content.
type cubeFile struct {
	name    string
	content []byte
	offset  int64
	mode    fs.FileMode
	modTime time.Time
}

func (file *cubeFile) Stat() (fs.FileInfo, error) {
	return coreio.NewFileInfo(file.name, int64(len(file.content)), file.mode, file.modTime, false), nil
}

func (file *cubeFile) Read(buffer []byte) (int, error) {
	if file.offset >= int64(len(file.content)) {
		return 0, goio.EOF
	}
	readCount := copy(buffer, file.content[file.offset:])
	file.offset += int64(readCount)
	return readCount, nil
}

func (file *cubeFile) Close() error {
	return nil
}

// cubeWriteCloser buffers writes and commits them (encrypted) on Close.
type cubeWriteCloser struct {
	medium *Medium
	path   string
	data   []byte
	mode   fs.FileMode
}

func (writer *cubeWriteCloser) Write(data []byte) (int, error) {
	writer.data = append(writer.data, data...)
	return len(data), nil
}

func (writer *cubeWriteCloser) Close() error {
	mode := writer.mode
	if mode == 0 {
		mode = 0644
	}
	return writer.medium.WriteMode(writer.path, string(writer.data), mode)
}

// AX-6-exception: core.NewBuffer is unavailable in the pinned core module; this is
// the minimal intrinsic writer needed by archive/tar.
type cubeArchiveBuffer struct {
	data []byte
}

func (buffer *cubeArchiveBuffer) Write(data []byte) (int, error) {
	buffer.data = append(buffer.data, data...)
	return len(data), nil
}

// Example: _ = cube.Pack("app.cube", workspaceMedium, key)
//
// Pack walks the source Medium, packs every file into a tar archive, encrypts
// the archive, and writes the ciphertext to outputPath on the local filesystem.
func Pack(outputPath string, source coreio.Medium, key []byte) error {
	if source == nil {
		return core.E(opCubePack, "source medium is required", fs.ErrInvalid)
	}
	if outputPath == "" {
		return core.E(opCubePack, "output path is required", fs.ErrInvalid)
	}

	archiveBytes, err := archiveMediumToTar(source)
	if err != nil {
		return core.E(opCubePack, "failed to build archive", err)
	}

	cipherSigil, err := sigil.NewChaChaPolySigil(key, nil)
	if err != nil {
		return core.E(opCubePack, errCreateCipher, err)
	}
	ciphertext, err := sigil.Transmute(archiveBytes, []sigil.Sigil{cipherSigil})
	if err != nil {
		return core.E(opCubePack, "failed to encrypt archive", err)
	}

	localMedium, relativePath, err := sandboxedLocalForPath(opCubePack, outputPath)
	if err != nil {
		return err
	}
	return localMedium.WriteMode(relativePath, string(ciphertext), 0600)
}

// Example: _ = cube.Unpack("app.cube", destinationMedium, key)
//
// Unpack reads the encrypted archive from cubePath, decrypts it, unpacks the
// tar contents, and writes every entry to the destination Medium.
func Unpack(cubePath string, destination coreio.Medium, key []byte) error {
	if destination == nil {
		return core.E(opCubeUnpack, "destination medium is required", fs.ErrInvalid)
	}
	if cubePath == "" {
		return core.E(opCubeUnpack, "cube path is required", fs.ErrInvalid)
	}

	localMedium, relativePath, err := sandboxedLocalForPath(opCubeUnpack, cubePath)
	if err != nil {
		return err
	}
	ciphertext, err := localMedium.Read(relativePath)
	if err != nil {
		return core.E(opCubeUnpack, core.Concat("failed to read cube: ", cubePath), err)
	}

	cipherSigil, err := sigil.NewChaChaPolySigil(key, nil)
	if err != nil {
		return core.E(opCubeUnpack, errCreateCipher, err)
	}
	archiveBytes, err := sigil.Untransmute([]byte(ciphertext), []sigil.Sigil{cipherSigil})
	if err != nil {
		return core.E(opCubeUnpack, "failed to decrypt archive", err)
	}

	return extractTarToMedium(archiveBytes, destination)
}

// Example: medium, _ := cube.Open("app.cube", key)
// Example: content, _ := medium.Read("config/app.yaml")
//
// Open reads the encrypted archive at cubePath, decrypts it, and returns a
// Medium backed by an in-memory node.Node. Reads and writes do not flow back
// to the .cube file — use Pack again to persist updates.
func Open(cubePath string, key []byte) (coreio.Medium, error) {
	if cubePath == "" {
		return nil, core.E(opCubeOpen, "cube path is required", fs.ErrInvalid)
	}

	localMedium, relativePath, err := sandboxedLocalForPath(opCubeOpen, cubePath)
	if err != nil {
		return nil, err
	}
	ciphertext, err := localMedium.Read(relativePath)
	if err != nil {
		return nil, core.E(opCubeOpen, core.Concat("failed to read cube: ", cubePath), err)
	}

	cipherSigil, err := sigil.NewChaChaPolySigil(key, nil)
	if err != nil {
		return nil, core.E(opCubeOpen, errCreateCipher, err)
	}
	archiveBytes, err := sigil.Untransmute([]byte(ciphertext), []sigil.Sigil{cipherSigil})
	if err != nil {
		return nil, core.E(opCubeOpen, "failed to decrypt archive", err)
	}

	nodeTree, err := node.FromTar(archiveBytes)
	if err != nil {
		return nil, core.E(opCubeOpen, "failed to load archive", err)
	}
	return nodeTree, nil
}

func sandboxedLocalForPath(operation, filePath string) (coreio.Medium, string, error) {
	if filePath == "" {
		return nil, "", core.E(operation, "path is required", fs.ErrInvalid)
	}
	if !core.PathIsAbs(filePath) {
		medium, err := coreio.NewSandboxed(".")
		if err != nil {
			return nil, "", core.E(operation, "failed to access local filesystem", err)
		}
		return medium, filePath, nil
	}
	root := core.PathDir(filePath)
	relativePath := core.PathBase(filePath)
	if root == "/" || relativePath == "" || relativePath == "." || relativePath == "/" {
		return nil, "", core.E(operation, core.Concat("invalid local path: ", filePath), fs.ErrInvalid)
	}
	medium, err := coreio.NewSandboxed(root)
	if err != nil {
		return nil, "", core.E(operation, "failed to access local filesystem", err)
	}
	return medium, relativePath, nil
}

// archiveMediumToTar walks source and serialises all files into a tar archive.
func archiveMediumToTar(source coreio.Medium) ([]byte, error) {
	buffer := &cubeArchiveBuffer{}
	tarWriter := tar.NewWriter(buffer)

	if err := walkAndArchive(source, "", tarWriter); err != nil {
		tarWriter.Close()
		return nil, err
	}

	if err := tarWriter.Close(); err != nil {
		return nil, core.E(opCubeArchive, "failed to close tar writer", err)
	}
	return buffer.data, nil
}

// walkAndArchive recursively walks the source and appends every file.
func walkAndArchive(source coreio.Medium, path string, tarWriter *tar.Writer) error {
	entries, err := source.List(path)
	if err != nil {
		return core.E(opCubeArchive, core.Concat("failed to list: ", path), err)
	}
	for _, entry := range entries {
		childPath := archiveChildPath(path, entry.Name())
		if entry.IsDir() {
			if err := walkAndArchive(source, childPath, tarWriter); err != nil {
				return err
			}
			continue
		}
		if err := writeTarFileEntry(source, childPath, tarWriter); err != nil {
			return err
		}
	}
	return nil
}

func archiveChildPath(parent, name string) string {
	if parent == "" {
		return name
	}
	return core.Concat(parent, "/", name)
}

func writeTarFileEntry(source coreio.Medium, filePath string, tarWriter *tar.Writer) error {
	content, err := source.Read(filePath)
	if err != nil {
		return core.E(opCubeArchive, core.Concat("failed to read: ", filePath), err)
	}
	mode, modTime := archiveEntryMetadata(source, filePath)
	header := &tar.Header{
		Name:    filePath,
		Mode:    int64(mode.Perm()),
		Size:    int64(len(content)),
		ModTime: modTime,
	}
	if err := tarWriter.WriteHeader(header); err != nil {
		return core.E(opCubeArchive, core.Concat("failed to write header: ", filePath), err)
	}
	if _, err := tarWriter.Write([]byte(content)); err != nil {
		return core.E(opCubeArchive, core.Concat("failed to write content: ", filePath), err)
	}
	return nil
}

func archiveEntryMetadata(source coreio.Medium, filePath string) (fs.FileMode, time.Time) {
	if info, err := source.Stat(filePath); err == nil {
		return info.Mode(), info.ModTime()
	}
	return 0600, time.Now()
}

// extractTarToMedium reads a tar archive and writes each entry to destination.
func extractTarToMedium(archiveBytes []byte, destination coreio.Medium) error {
	tarReader := tar.NewReader(&cubeFile{content: archiveBytes})
	for {
		header, err := tarReader.Next()
		if err == goio.EOF {
			return nil
		}
		if err != nil {
			return core.E(opCubeExtract, "failed to read tar entry", err)
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}
		if err := extractTarFileEntry(tarReader, header, destination); err != nil {
			return err
		}
	}
}

func extractTarFileEntry(tarReader *tar.Reader, header *tar.Header, destination coreio.Medium) error {
	content, err := readTarEntryContent(tarReader, header.Name)
	if err != nil {
		return err
	}
	name, ok, err := validatedTarEntryName(header.Name)
	if err != nil || !ok {
		return err
	}
	mode := fs.FileMode(header.Mode)
	if mode == 0 {
		mode = 0644
	}
	if err := destination.WriteMode(name, content, mode); err != nil {
		return core.E(opCubeExtract, core.Concat("failed to write entry: ", name), err)
	}
	return nil
}

func readTarEntryContent(tarReader *tar.Reader, name string) (string, error) {
	contentResult := core.ReadAll(tarReader)
	if contentResult.OK {
		content, ok := contentResult.Value.(string)
		if !ok {
			return "", core.E(opCubeExtract, core.Concat("failed to read entry: ", name), fs.ErrInvalid)
		}
		return content, nil
	}
	if err, ok := contentResult.Value.(error); ok {
		return "", core.E(opCubeExtract, core.Concat("failed to read entry: ", name), err)
	}
	return "", core.E(opCubeExtract, core.Concat("failed to read entry: ", name), fs.ErrInvalid)
}

func validatedTarEntryName(rawName string) (string, bool, error) {
	if rawName == "" {
		return "", false, nil
	}
	if path.IsAbs(rawName) || core.Contains(rawName, "\\") {
		return "", false, core.E(opCubeExtract, core.Concat("invalid tar entry path: ", rawName), fs.ErrInvalid)
	}
	name := core.TrimPrefix(rawName, "/")
	if name == "" || core.HasSuffix(name, "/") {
		return "", false, nil
	}
	if hasParentPathSegment(name) {
		return "", false, core.E(opCubeExtract, core.Concat("invalid tar entry path: ", name), fs.ErrInvalid)
	}
	clean := path.Clean(name)
	if clean == "." || clean == "" || clean == ".." || core.HasPrefix(clean, "../") {
		return "", false, core.E(opCubeExtract, core.Concat("invalid tar entry path: ", name), fs.ErrInvalid)
	}
	return clean, true, nil
}

func hasParentPathSegment(name string) bool {
	for _, part := range core.Split(name, "/") {
		if part == ".." {
			return true
		}
	}
	return false
}
