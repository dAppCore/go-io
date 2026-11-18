# API Contract

Descriptions use doc comments when present; otherwise they are short code-based summaries.
Test coverage is `Yes` when same-package tests directly execute or reference the exported symbol; otherwise `No`.
`CODEX.md` was not present in the repository at generation time.

| Name | Signature | Package Path | Description | Test Coverage |
| --- | --- | --- | --- | --- |
| `DirEntry` | `type DirEntry struct` | `github.com/dappcore/go-io/go` | DirEntry provides a simple implementation of fs.DirEntry for mock testing. | Yes |
| `FileInfo` | `type FileInfo struct` | `github.com/dappcore/go-io/go` | FileInfo provides a simple implementation of fs.FileInfo for mock testing. | Yes |
| `Medium` | `type Medium interface` | `github.com/dappcore/go-io/go` | Medium defines the standard interface for a storage backend. | Yes |
| `MockFile` | `type MockFile struct` | `github.com/dappcore/go-io/go` | MockFile implements fs.File for MockMedium. | No |
| `MockMedium` | `type MockMedium struct` | `github.com/dappcore/go-io/go` | MockMedium is an in-memory implementation of Medium for testing. | Yes |
| `MockWriteCloser` | `type MockWriteCloser struct` | `github.com/dappcore/go-io/go` | MockWriteCloser implements WriteCloser for MockMedium. | No |
| `Copy` | `func Copy(src Medium, srcPath string, dst Medium, dstPath string) error` | `github.com/dappcore/go-io/go` | Copy copies a file from one medium to another. | Yes |
| `EnsureDir` | `func EnsureDir(m Medium, path string) error` | `github.com/dappcore/go-io/go` | EnsureDir makes sure a directory exists in the given medium. | Yes |
| `IsFile` | `func IsFile(m Medium, path string) bool` | `github.com/dappcore/go-io/go` | IsFile checks if a path exists and is a regular file in the given medium. | Yes |
| `NewMockMedium` | `func NewMockMedium() *MockMedium` | `github.com/dappcore/go-io/go` | NewMockMedium creates a new MockMedium instance. | Yes |
| `NewSandboxed` | `func NewSandboxed(root string) (Medium, error)` | `github.com/dappcore/go-io/go` | NewSandboxed creates a new Medium sandboxed to the given root directory. | No |
| `Read` | `func Read(m Medium, path string) (string, error)` | `github.com/dappcore/go-io/go` | Read retrieves the content of a file from the given medium. | Yes |
| `ReadStream` | `func ReadStream(m Medium, path string) (goio.ReadCloser, error)` | `github.com/dappcore/go-io/go` | ReadStream returns a reader for the file content from the given medium. | No |
| `Write` | `func Write(m Medium, path, content string) error` | `github.com/dappcore/go-io/go` | Write saves the given content to a file in the given medium. | Yes |
| `WriteStream` | `func WriteStream(m Medium, path string) (goio.WriteCloser, error)` | `github.com/dappcore/go-io/go` | WriteStream returns a writer for the file content in the given medium. | No |
| `DirEntry.Info` | `func (DirEntry) Info() (fs.FileInfo, error)` | `github.com/dappcore/go-io/go` | Returns file info for the entry. | No |
| `DirEntry.IsDir` | `func (DirEntry) IsDir() bool` | `github.com/dappcore/go-io/go` | Reports whether the entry represents a directory. | No |
| `DirEntry.Name` | `func (DirEntry) Name() string` | `github.com/dappcore/go-io/go` | Returns the stored entry name. | Yes |
| `DirEntry.Type` | `func (DirEntry) Type() fs.FileMode` | `github.com/dappcore/go-io/go` | Returns the entry type bits. | No |
| `FileInfo.IsDir` | `func (FileInfo) IsDir() bool` | `github.com/dappcore/go-io/go` | Reports whether the entry represents a directory. | Yes |
| `FileInfo.ModTime` | `func (FileInfo) ModTime() time.Time` | `github.com/dappcore/go-io/go` | Returns the stored modification time. | No |
| `FileInfo.Mode` | `func (FileInfo) Mode() fs.FileMode` | `github.com/dappcore/go-io/go` | Returns the stored file mode. | No |
| `FileInfo.Name` | `func (FileInfo) Name() string` | `github.com/dappcore/go-io/go` | Returns the stored entry name. | Yes |
| `FileInfo.Size` | `func (FileInfo) Size() int64` | `github.com/dappcore/go-io/go` | Returns the stored size in bytes. | Yes |
| `FileInfo.Sys` | `func (FileInfo) Sys() any` | `github.com/dappcore/go-io/go` | Returns the underlying system-specific data. | No |
| `Medium.Append` | `Append(path string) (goio.WriteCloser, error)` | `github.com/dappcore/go-io/go` | Append opens the named file for appending, creating it if it doesn't exist. | No |
| `Medium.Create` | `Create(path string) (goio.WriteCloser, error)` | `github.com/dappcore/go-io/go` | Create creates or truncates the named file. | No |
| `Medium.Delete` | `Delete(path string) error` | `github.com/dappcore/go-io/go` | Delete removes a file or empty directory. | Yes |
| `Medium.DeleteAll` | `DeleteAll(path string) error` | `github.com/dappcore/go-io/go` | DeleteAll removes a file or directory and all its contents recursively. | Yes |
| `Medium.EnsureDir` | `EnsureDir(path string) error` | `github.com/dappcore/go-io/go` | EnsureDir makes sure a directory exists, creating it if necessary. | Yes |
| `Medium.Exists` | `Exists(path string) bool` | `github.com/dappcore/go-io/go` | Exists checks if a path exists (file or directory). | Yes |
| `Medium.FileGet` | `FileGet(path string) (string, error)` | `github.com/dappcore/go-io/go` | FileGet is a convenience function that reads a file from the medium. | Yes |
| `Medium.FileSet` | `FileSet(path, content string) error` | `github.com/dappcore/go-io/go` | FileSet is a convenience function that writes a file to the medium. | Yes |
| `Medium.IsDir` | `IsDir(path string) bool` | `github.com/dappcore/go-io/go` | IsDir checks if a path exists and is a directory. | Yes |
| `Medium.IsFile` | `IsFile(path string) bool` | `github.com/dappcore/go-io/go` | IsFile checks if a path exists and is a regular file. | Yes |
| `Medium.List` | `List(path string) ([]fs.DirEntry, error)` | `github.com/dappcore/go-io/go` | List returns the directory entries for the given path. | Yes |
| `Medium.Open` | `Open(path string) (fs.File, error)` | `github.com/dappcore/go-io/go` | Open opens the named file for reading. | No |
| `Medium.Read` | `Read(path string) (string, error)` | `github.com/dappcore/go-io/go` | Read retrieves the content of a file as a string. | Yes |
| `Medium.ReadStream` | `ReadStream(path string) (goio.ReadCloser, error)` | `github.com/dappcore/go-io/go` | ReadStream returns a reader for the file content. | No |
| `Medium.Rename` | `Rename(oldPath, newPath string) error` | `github.com/dappcore/go-io/go` | Rename moves a file or directory from oldPath to newPath. | Yes |
| `Medium.Stat` | `Stat(path string) (fs.FileInfo, error)` | `github.com/dappcore/go-io/go` | Stat returns file information for the given path. | Yes |
| `Medium.Write` | `Write(path, content string) error` | `github.com/dappcore/go-io/go` | Write saves the given content to a file, overwriting it if it exists. | Yes |
| `Medium.WriteMode` | `WriteMode(path, content string, mode os.FileMode) error` | `github.com/dappcore/go-io/go` | WriteMode saves content with explicit file permissions. | No |
| `Medium.WriteStream` | `WriteStream(path string) (goio.WriteCloser, error)` | `github.com/dappcore/go-io/go` | WriteStream returns a writer for the file content. | No |
| `MockFile.Close` | `func (*MockFile) Close() error` | `github.com/dappcore/go-io/go` | Closes the current value. | No |
| `MockFile.Read` | `func (*MockFile) Read(b []byte) (int, error)` | `github.com/dappcore/go-io/go` | Reads data from the current value. | No |
| `MockFile.Stat` | `func (*MockFile) Stat() (fs.FileInfo, error)` | `github.com/dappcore/go-io/go` | Returns file metadata for the current value. | No |
| `MockMedium.Append` | `func (*MockMedium) Append(path string) (goio.WriteCloser, error)` | `github.com/dappcore/go-io/go` | Append opens a file for appending in the mock filesystem. | No |
| `MockMedium.Create` | `func (*MockMedium) Create(path string) (goio.WriteCloser, error)` | `github.com/dappcore/go-io/go` | Create creates a file in the mock filesystem. | No |
| `MockMedium.Delete` | `func (*MockMedium) Delete(path string) error` | `github.com/dappcore/go-io/go` | Delete removes a file or empty directory from the mock filesystem. | Yes |
| `MockMedium.DeleteAll` | `func (*MockMedium) DeleteAll(path string) error` | `github.com/dappcore/go-io/go` | DeleteAll removes a file or directory and all contents from the mock filesystem. | Yes |
| `MockMedium.EnsureDir` | `func (*MockMedium) EnsureDir(path string) error` | `github.com/dappcore/go-io/go` | EnsureDir records that a directory exists in the mock filesystem. | Yes |
| `MockMedium.Exists` | `func (*MockMedium) Exists(path string) bool` | `github.com/dappcore/go-io/go` | Exists checks if a path exists in the mock filesystem. | Yes |
| `MockMedium.FileGet` | `func (*MockMedium) FileGet(path string) (string, error)` | `github.com/dappcore/go-io/go` | FileGet is a convenience function that reads a file from the mock filesystem. | Yes |
| `MockMedium.FileSet` | `func (*MockMedium) FileSet(path, content string) error` | `github.com/dappcore/go-io/go` | FileSet is a convenience function that writes a file to the mock filesystem. | Yes |
| `MockMedium.IsDir` | `func (*MockMedium) IsDir(path string) bool` | `github.com/dappcore/go-io/go` | IsDir checks if a path is a directory in the mock filesystem. | Yes |
| `MockMedium.IsFile` | `func (*MockMedium) IsFile(path string) bool` | `github.com/dappcore/go-io/go` | IsFile checks if a path exists as a file in the mock filesystem. | Yes |
| `MockMedium.List` | `func (*MockMedium) List(path string) ([]fs.DirEntry, error)` | `github.com/dappcore/go-io/go` | List returns directory entries for the mock filesystem. | Yes |
| `MockMedium.Open` | `func (*MockMedium) Open(path string) (fs.File, error)` | `github.com/dappcore/go-io/go` | Open opens a file from the mock filesystem. | No |
| `MockMedium.Read` | `func (*MockMedium) Read(path string) (string, error)` | `github.com/dappcore/go-io/go` | Read retrieves the content of a file from the mock filesystem. | Yes |
| `MockMedium.ReadStream` | `func (*MockMedium) ReadStream(path string) (goio.ReadCloser, error)` | `github.com/dappcore/go-io/go` | ReadStream returns a reader for the file content in the mock filesystem. | No |
| `MockMedium.Rename` | `func (*MockMedium) Rename(oldPath, newPath string) error` | `github.com/dappcore/go-io/go` | Rename moves a file or directory in the mock filesystem. | Yes |
| `MockMedium.Stat` | `func (*MockMedium) Stat(path string) (fs.FileInfo, error)` | `github.com/dappcore/go-io/go` | Stat returns file information for the mock filesystem. | Yes |
| `MockMedium.Write` | `func (*MockMedium) Write(path, content string) error` | `github.com/dappcore/go-io/go` | Write saves the given content to a file in the mock filesystem. | Yes |
| `MockMedium.WriteMode` | `func (*MockMedium) WriteMode(path, content string, mode os.FileMode) error` | `github.com/dappcore/go-io/go` | Writes content using an explicit file mode. | No |
| `MockMedium.WriteStream` | `func (*MockMedium) WriteStream(path string) (goio.WriteCloser, error)` | `github.com/dappcore/go-io/go` | WriteStream returns a writer for the file content in the mock filesystem. | No |
| `MockWriteCloser.Close` | `func (*MockWriteCloser) Close() error` | `github.com/dappcore/go-io/go` | Closes the current value. | No |
| `MockWriteCloser.Write` | `func (*MockWriteCloser) Write(p []byte) (int, error)` | `github.com/dappcore/go-io/go` | Writes data to the current value. | No |
| `Medium` | `type Medium struct` | `github.com/dappcore/go-io/go/datanode` | Medium is an in-memory storage backend backed by a Borg DataNode. | Yes |
| `FromTar` | `func FromTar(data []byte) (*Medium, error)` | `github.com/dappcore/go-io/go/datanode` | FromTar creates a Medium from a tarball, restoring all files. | Yes |
| `New` | `func New() *Medium` | `github.com/dappcore/go-io/go/datanode` | New creates a new empty DataNode Medium. | Yes |
| `Medium.Append` | `func (*Medium) Append(p string) (goio.WriteCloser, error)` | `github.com/dappcore/go-io/go/datanode` | Opens the named file for appending, creating it if needed. | Yes |
| `Medium.Create` | `func (*Medium) Create(p string) (goio.WriteCloser, error)` | `github.com/dappcore/go-io/go/datanode` | Creates or truncates the named file and returns a writer. | Yes |
| `Medium.DataNode` | `func (*Medium) DataNode() *datanode.DataNode` | `github.com/dappcore/go-io/go/datanode` | DataNode returns the underlying Borg DataNode. | Yes |
| `Medium.Delete` | `func (*Medium) Delete(p string) error` | `github.com/dappcore/go-io/go/datanode` | Removes a file, key, or empty directory. | Yes |
| `Medium.DeleteAll` | `func (*Medium) DeleteAll(p string) error` | `github.com/dappcore/go-io/go/datanode` | Removes a file or directory tree recursively. | Yes |
| `Medium.EnsureDir` | `func (*Medium) EnsureDir(p string) error` | `github.com/dappcore/go-io/go/datanode` | Ensures a directory path exists. | Yes |
| `Medium.Exists` | `func (*Medium) Exists(p string) bool` | `github.com/dappcore/go-io/go/datanode` | Reports whether the path exists. | Yes |
| `Medium.FileGet` | `func (*Medium) FileGet(p string) (string, error)` | `github.com/dappcore/go-io/go/datanode` | Reads a file or key through the convenience accessor. | Yes |
| `Medium.FileSet` | `func (*Medium) FileSet(p, content string) error` | `github.com/dappcore/go-io/go/datanode` | Writes a file or key through the convenience accessor. | Yes |
| `Medium.IsDir` | `func (*Medium) IsDir(p string) bool` | `github.com/dappcore/go-io/go/datanode` | Reports whether the entry represents a directory. | Yes |
| `Medium.IsFile` | `func (*Medium) IsFile(p string) bool` | `github.com/dappcore/go-io/go/datanode` | Reports whether the path exists as a regular file. | Yes |
| `Medium.List` | `func (*Medium) List(p string) ([]fs.DirEntry, error)` | `github.com/dappcore/go-io/go/datanode` | Lists directory entries beneath the given path. | Yes |
| `Medium.Open` | `func (*Medium) Open(p string) (fs.File, error)` | `github.com/dappcore/go-io/go/datanode` | Opens the named file for reading. | Yes |
| `Medium.Read` | `func (*Medium) Read(p string) (string, error)` | `github.com/dappcore/go-io/go/datanode` | Reads data from the current value. | Yes |
| `Medium.ReadStream` | `func (*Medium) ReadStream(p string) (goio.ReadCloser, error)` | `github.com/dappcore/go-io/go/datanode` | Opens a streaming reader for the file content. | Yes |
| `Medium.Rename` | `func (*Medium) Rename(oldPath, newPath string) error` | `github.com/dappcore/go-io/go/datanode` | Moves a file or directory to a new path. | Yes |
| `Medium.Restore` | `func (*Medium) Restore(data []byte) error` | `github.com/dappcore/go-io/go/datanode` | Restore replaces the filesystem contents from a tarball. | Yes |
| `Medium.Snapshot` | `func (*Medium) Snapshot() ([]byte, error)` | `github.com/dappcore/go-io/go/datanode` | Snapshot serializes the entire filesystem to a tarball. | Yes |
| `Medium.Stat` | `func (*Medium) Stat(p string) (fs.FileInfo, error)` | `github.com/dappcore/go-io/go/datanode` | Returns file metadata for the current value. | Yes |
| `Medium.Write` | `func (*Medium) Write(p, content string) error` | `github.com/dappcore/go-io/go/datanode` | Writes data to the current value. | Yes |
| `Medium.WriteMode` | `func (*Medium) WriteMode(p, content string, mode os.FileMode) error` | `github.com/dappcore/go-io/go/datanode` | Writes content using an explicit file mode. | No |
| `Medium.WriteStream` | `func (*Medium) WriteStream(p string) (goio.WriteCloser, error)` | `github.com/dappcore/go-io/go/datanode` | Opens a streaming writer for the file content. | Yes |
| `Medium` | `type Medium struct` | `github.com/dappcore/go-io/go/local` | Medium is a local filesystem storage backend. | Yes |
| `New` | `func New(root string) (*Medium, error)` | `github.com/dappcore/go-io/go/local` | New creates a new local Medium rooted at the given directory. | Yes |
| `Medium.Append` | `func (*Medium) Append(p string) (goio.WriteCloser, error)` | `github.com/dappcore/go-io/go/local` | Append opens the named file for appending, creating it if it doesn't exist. | No |
| `Medium.Create` | `func (*Medium) Create(p string) (goio.WriteCloser, error)` | `github.com/dappcore/go-io/go/local` | Create creates or truncates the named file. | Yes |
| `Medium.Delete` | `func (*Medium) Delete(p string) error` | `github.com/dappcore/go-io/go/local` | Delete removes a file or empty directory. | Yes |
| `Medium.DeleteAll` | `func (*Medium) DeleteAll(p string) error` | `github.com/dappcore/go-io/go/local` | DeleteAll removes a file or directory recursively. | Yes |
| `Medium.EnsureDir` | `func (*Medium) EnsureDir(p string) error` | `github.com/dappcore/go-io/go/local` | EnsureDir creates directory if it doesn't exist. | Yes |
| `Medium.Exists` | `func (*Medium) Exists(p string) bool` | `github.com/dappcore/go-io/go/local` | Exists returns true if path exists. | Yes |
| `Medium.FileGet` | `func (*Medium) FileGet(p string) (string, error)` | `github.com/dappcore/go-io/go/local` | FileGet is an alias for Read. | Yes |
| `Medium.FileSet` | `func (*Medium) FileSet(p, content string) error` | `github.com/dappcore/go-io/go/local` | FileSet is an alias for Write. | Yes |
| `Medium.IsDir` | `func (*Medium) IsDir(p string) bool` | `github.com/dappcore/go-io/go/local` | IsDir returns true if path is a directory. | Yes |
| `Medium.IsFile` | `func (*Medium) IsFile(p string) bool` | `github.com/dappcore/go-io/go/local` | IsFile returns true if path is a regular file. | Yes |
| `Medium.List` | `func (*Medium) List(p string) ([]fs.DirEntry, error)` | `github.com/dappcore/go-io/go/local` | List returns directory entries. | Yes |
| `Medium.Open` | `func (*Medium) Open(p string) (fs.File, error)` | `github.com/dappcore/go-io/go/local` | Open opens the named file for reading. | Yes |
| `Medium.Read` | `func (*Medium) Read(p string) (string, error)` | `github.com/dappcore/go-io/go/local` | Read returns file contents as string. | Yes |
| `Medium.ReadStream` | `func (*Medium) ReadStream(path string) (goio.ReadCloser, error)` | `github.com/dappcore/go-io/go/local` | ReadStream returns a reader for the file content. | Yes |
| `Medium.Rename` | `func (*Medium) Rename(oldPath, newPath string) error` | `github.com/dappcore/go-io/go/local` | Rename moves a file or directory. | Yes |
| `Medium.Stat` | `func (*Medium) Stat(p string) (fs.FileInfo, error)` | `github.com/dappcore/go-io/go/local` | Stat returns file info. | Yes |
| `Medium.Write` | `func (*Medium) Write(p, content string) error` | `github.com/dappcore/go-io/go/local` | Write saves content to file, creating parent directories as needed. | Yes |
| `Medium.WriteMode` | `func (*Medium) WriteMode(p, content string, mode os.FileMode) error` | `github.com/dappcore/go-io/go/local` | WriteMode saves content to file with explicit permissions. | Yes |
| `Medium.WriteStream` | `func (*Medium) WriteStream(path string) (goio.WriteCloser, error)` | `github.com/dappcore/go-io/go/local` | WriteStream returns a writer for the file content. | Yes |
| `Node` | `type Node struct` | `github.com/dappcore/go-io/go/node` | Node is an in-memory filesystem that implements coreio.Node (and therefore coreio.Medium). | Yes |
| `WalkOptions` | `type WalkOptions struct` | `github.com/dappcore/go-io/go/node` | WalkOptions configures the behaviour of Walk. | Yes |
| `FromTar` | `func FromTar(data []byte) (*Node, error)` | `github.com/dappcore/go-io/go/node` | FromTar creates a new Node from a tar archive. | Yes |
| `New` | `func New() *Node` | `github.com/dappcore/go-io/go/node` | New creates a new, empty Node. | Yes |
| `Node.AddData` | `func (*Node) AddData(name string, content []byte)` | `github.com/dappcore/go-io/go/node` | AddData stages content in the in-memory filesystem. | Yes |
| `Node.Append` | `func (*Node) Append(p string) (goio.WriteCloser, error)` | `github.com/dappcore/go-io/go/node` | Append opens the named file for appending, creating it if needed. | No |
| `Node.ExportFile` | `func (*Node) ExportFile(src, dst string, perm fs.FileMode) error` | `github.com/dappcore/go-io/go/node` | ExportFile exports a file from the in-memory tree to the local filesystem. Use CopyTo for Medium-agnostic transfers. | Yes |
| `Node.CopyTo` | `func (*Node) CopyTo(target coreio.Medium, sourcePath, destPath string) error` | `github.com/dappcore/go-io/go/node` | CopyTo copies a file (or directory tree) from the node to any Medium. | No |
| `Node.Create` | `func (*Node) Create(p string) (goio.WriteCloser, error)` | `github.com/dappcore/go-io/go/node` | Create creates or truncates the named file, returning a WriteCloser. | No |
| `Node.Delete` | `func (*Node) Delete(p string) error` | `github.com/dappcore/go-io/go/node` | Delete removes a single file. | No |
| `Node.DeleteAll` | `func (*Node) DeleteAll(p string) error` | `github.com/dappcore/go-io/go/node` | DeleteAll removes a file or directory and all children. | No |
| `Node.EnsureDir` | `func (*Node) EnsureDir(_ string) error` | `github.com/dappcore/go-io/go/node` | EnsureDir is a no-op because directories are implicit in Node. | No |
| `Node.Exists` | `func (*Node) Exists(p string) bool` | `github.com/dappcore/go-io/go/node` | Exists checks if a path exists (file or directory). | Yes |
| `Node.FileGet` | `func (*Node) FileGet(p string) (string, error)` | `github.com/dappcore/go-io/go/node` | FileGet is an alias for Read. | No |
| `Node.FileSet` | `func (*Node) FileSet(p, content string) error` | `github.com/dappcore/go-io/go/node` | FileSet is an alias for Write. | No |
| `Node.IsDir` | `func (*Node) IsDir(p string) bool` | `github.com/dappcore/go-io/go/node` | IsDir checks if a path exists and is a directory. | No |
| `Node.IsFile` | `func (*Node) IsFile(p string) bool` | `github.com/dappcore/go-io/go/node` | IsFile checks if a path exists and is a regular file. | No |
| `Node.List` | `func (*Node) List(p string) ([]fs.DirEntry, error)` | `github.com/dappcore/go-io/go/node` | List returns directory entries for the given path. | No |
| `Node.LoadTar` | `func (*Node) LoadTar(data []byte) error` | `github.com/dappcore/go-io/go/node` | LoadTar replaces the in-memory tree with the contents of a tar archive. | Yes |
| `Node.Open` | `func (*Node) Open(name string) (fs.File, error)` | `github.com/dappcore/go-io/go/node` | Open opens a file from the Node. | Yes |
| `Node.Read` | `func (*Node) Read(p string) (string, error)` | `github.com/dappcore/go-io/go/node` | Read retrieves the content of a file as a string. | No |
| `Node.ReadDir` | `func (*Node) ReadDir(name string) ([]fs.DirEntry, error)` | `github.com/dappcore/go-io/go/node` | ReadDir reads and returns all directory entries for the named directory. | Yes |
| `Node.ReadFile` | `func (*Node) ReadFile(name string) ([]byte, error)` | `github.com/dappcore/go-io/go/node` | ReadFile returns the content of the named file as a byte slice. | Yes |
| `Node.ReadStream` | `func (*Node) ReadStream(p string) (goio.ReadCloser, error)` | `github.com/dappcore/go-io/go/node` | ReadStream returns a ReadCloser for the file content. | No |
| `Node.Rename` | `func (*Node) Rename(oldPath, newPath string) error` | `github.com/dappcore/go-io/go/node` | Rename moves a file from oldPath to newPath. | No |
| `Node.Stat` | `func (*Node) Stat(name string) (fs.FileInfo, error)` | `github.com/dappcore/go-io/go/node` | Stat returns file information for the given path. | Yes |
| `Node.ToTar` | `func (*Node) ToTar() ([]byte, error)` | `github.com/dappcore/go-io/go/node` | ToTar serialises the entire in-memory tree to a tar archive. | Yes |
| `Node.Walk` | `func (*Node) Walk(root string, fn fs.WalkDirFunc, opts ...WalkOptions) error` | `github.com/dappcore/go-io/go/node` | Walk walks the in-memory tree with optional WalkOptions. | Yes |
| `Node.WalkNode` | `func (*Node) WalkNode(root string, fn fs.WalkDirFunc) error` | `github.com/dappcore/go-io/go/node` | WalkNode walks the in-memory tree, calling fn for each entry. | No |
| `Node.Write` | `func (*Node) Write(p, content string) error` | `github.com/dappcore/go-io/go/node` | Write saves the given content to a file, overwriting it if it exists. | No |
| `Node.WriteMode` | `func (*Node) WriteMode(p, content string, mode os.FileMode) error` | `github.com/dappcore/go-io/go/node` | WriteMode saves content with explicit permissions (no-op for in-memory node). | No |
| `Node.WriteStream` | `func (*Node) WriteStream(p string) (goio.WriteCloser, error)` | `github.com/dappcore/go-io/go/node` | WriteStream returns a WriteCloser for the file content. | No |
| `Medium` | `type Medium struct` | `github.com/dappcore/go-io/go/s3` | Medium is an S3-backed storage backend implementing the io.Medium interface. | Yes |
| `Option` | `type Option func(*Medium)` | `github.com/dappcore/go-io/go/s3` | Option configures a Medium. | Yes |
| `New` | `func New(bucket string, opts ...Option) (*Medium, error)` | `github.com/dappcore/go-io/go/s3` | New creates a new S3 Medium for the given bucket. | Yes |
| `WithClient` | `func WithClient(client *s3.Client) Option` | `github.com/dappcore/go-io/go/s3` | WithClient sets the S3 client for dependency injection. | No |
| `WithPrefix` | `func WithPrefix(prefix string) Option` | `github.com/dappcore/go-io/go/s3` | WithPrefix sets an optional key prefix for all operations. | Yes |
| `Medium.Append` | `func (*Medium) Append(p string) (goio.WriteCloser, error)` | `github.com/dappcore/go-io/go/s3` | Append opens the named file for appending. | Yes |
| `Medium.Create` | `func (*Medium) Create(p string) (goio.WriteCloser, error)` | `github.com/dappcore/go-io/go/s3` | Create creates or truncates the named file. | Yes |
| `Medium.Delete` | `func (*Medium) Delete(p string) error` | `github.com/dappcore/go-io/go/s3` | Delete removes a single object. | Yes |
| `Medium.DeleteAll` | `func (*Medium) DeleteAll(p string) error` | `github.com/dappcore/go-io/go/s3` | DeleteAll removes all objects under the given prefix. | Yes |
| `Medium.EnsureDir` | `func (*Medium) EnsureDir(_ string) error` | `github.com/dappcore/go-io/go/s3` | EnsureDir is a no-op for S3 (S3 has no real directories). | Yes |
| `Medium.Exists` | `func (*Medium) Exists(p string) bool` | `github.com/dappcore/go-io/go/s3` | Exists checks if a path exists (file or directory prefix). | Yes |
| `Medium.FileGet` | `func (*Medium) FileGet(p string) (string, error)` | `github.com/dappcore/go-io/go/s3` | FileGet is a convenience function that reads a file from the medium. | Yes |
| `Medium.FileSet` | `func (*Medium) FileSet(p, content string) error` | `github.com/dappcore/go-io/go/s3` | FileSet is a convenience function that writes a file to the medium. | Yes |
| `Medium.IsDir` | `func (*Medium) IsDir(p string) bool` | `github.com/dappcore/go-io/go/s3` | IsDir checks if a path exists and is a directory (has objects under it as a prefix). | Yes |
| `Medium.IsFile` | `func (*Medium) IsFile(p string) bool` | `github.com/dappcore/go-io/go/s3` | IsFile checks if a path exists and is a regular file (not a "directory" prefix). | Yes |
| `Medium.List` | `func (*Medium) List(p string) ([]fs.DirEntry, error)` | `github.com/dappcore/go-io/go/s3` | List returns directory entries for the given path using ListObjectsV2 with delimiter. | Yes |
| `Medium.Open` | `func (*Medium) Open(p string) (fs.File, error)` | `github.com/dappcore/go-io/go/s3` | Open opens the named file for reading. | Yes |
| `Medium.Read` | `func (*Medium) Read(p string) (string, error)` | `github.com/dappcore/go-io/go/s3` | Read retrieves the content of a file as a string. | Yes |
| `Medium.ReadStream` | `func (*Medium) ReadStream(p string) (goio.ReadCloser, error)` | `github.com/dappcore/go-io/go/s3` | ReadStream returns a reader for the file content. | Yes |
| `Medium.Rename` | `func (*Medium) Rename(oldPath, newPath string) error` | `github.com/dappcore/go-io/go/s3` | Rename moves an object by copying then deleting the original. | Yes |
| `Medium.Stat` | `func (*Medium) Stat(p string) (fs.FileInfo, error)` | `github.com/dappcore/go-io/go/s3` | Stat returns file information for the given path using HeadObject. | Yes |
| `Medium.Write` | `func (*Medium) Write(p, content string) error` | `github.com/dappcore/go-io/go/s3` | Write saves the given content to a file, overwriting it if it exists. | Yes |
| `Medium.WriteStream` | `func (*Medium) WriteStream(p string) (goio.WriteCloser, error)` | `github.com/dappcore/go-io/go/s3` | WriteStream returns a writer for the file content. | Yes |
| `Base64Sigil` | `type Base64Sigil struct` | `github.com/dappcore/go-io/go/sigil` | Base64Sigil is a Sigil that encodes/decodes data to/from base64. | Yes |
| `ChaChaPolySigil` | `type ChaChaPolySigil struct` | `github.com/dappcore/go-io/go/sigil` | ChaChaPolySigil is a Sigil that encrypts/decrypts data using ChaCha20-Poly1305. | Yes |
| `GzipSigil` | `type GzipSigil struct` | `github.com/dappcore/go-io/go/sigil` | GzipSigil is a Sigil that compresses/decompresses data using gzip. | Yes |
| `HashSigil` | `type HashSigil struct` | `github.com/dappcore/go-io/go/sigil` | HashSigil is a Sigil that hashes the data using a specified algorithm. | Yes |
| `HexSigil` | `type HexSigil struct` | `github.com/dappcore/go-io/go/sigil` | HexSigil is a Sigil that encodes/decodes data to/from hexadecimal. | Yes |
| `JSONSigil` | `type JSONSigil struct` | `github.com/dappcore/go-io/go/sigil` | JSONSigil is a Sigil that compacts or indents JSON data. | Yes |
| `PreObfuscator` | `type PreObfuscator interface` | `github.com/dappcore/go-io/go/sigil` | PreObfuscator applies a reversible transformation to data before encryption. | Yes |
| `ReverseSigil` | `type ReverseSigil struct` | `github.com/dappcore/go-io/go/sigil` | ReverseSigil is a Sigil that reverses the bytes of the payload. | Yes |
| `ShuffleMaskObfuscator` | `type ShuffleMaskObfuscator struct` | `github.com/dappcore/go-io/go/sigil` | ShuffleMaskObfuscator provides stronger obfuscation through byte shuffling and masking. | Yes |
| `Sigil` | `type Sigil interface` | `github.com/dappcore/go-io/go/sigil` | Sigil defines the interface for a data transformer. | Yes |
| `XORObfuscator` | `type XORObfuscator struct` | `github.com/dappcore/go-io/go/sigil` | XORObfuscator performs XOR-based obfuscation using an entropy-derived key stream. | Yes |
| `GetNonceFromCiphertext` | `func GetNonceFromCiphertext(ciphertext []byte) ([]byte, error)` | `github.com/dappcore/go-io/go/sigil` | GetNonceFromCiphertext extracts the nonce from encrypted output. | Yes |
| `NewChaChaPolySigil` | `func NewChaChaPolySigil(key []byte) (*ChaChaPolySigil, error)` | `github.com/dappcore/go-io/go/sigil` | NewChaChaPolySigil creates a new encryption sigil with the given key. | Yes |
| `NewChaChaPolySigilWithObfuscator` | `func NewChaChaPolySigilWithObfuscator(key []byte, obfuscator PreObfuscator) (*ChaChaPolySigil, error)` | `github.com/dappcore/go-io/go/sigil` | NewChaChaPolySigilWithObfuscator creates a new encryption sigil with custom obfuscator. | Yes |
| `NewHashSigil` | `func NewHashSigil(h crypto.Hash) *HashSigil` | `github.com/dappcore/go-io/go/sigil` | NewHashSigil creates a new HashSigil. | Yes |
| `NewSigil` | `func NewSigil(name string) (Sigil, error)` | `github.com/dappcore/go-io/go/sigil` | NewSigil is a factory function that returns a Sigil based on a string name. | Yes |
| `Transmute` | `func Transmute(data []byte, sigils []Sigil) ([]byte, error)` | `github.com/dappcore/go-io/go/sigil` | Transmute applies a series of sigils to data in sequence. | Yes |
| `Untransmute` | `func Untransmute(data []byte, sigils []Sigil) ([]byte, error)` | `github.com/dappcore/go-io/go/sigil` | Untransmute reverses a transmutation by applying Out in reverse order. | Yes |
| `Base64Sigil.In` | `func (*Base64Sigil) In(data []byte) ([]byte, error)` | `github.com/dappcore/go-io/go/sigil` | In encodes the data to base64. | Yes |
| `Base64Sigil.Out` | `func (*Base64Sigil) Out(data []byte) ([]byte, error)` | `github.com/dappcore/go-io/go/sigil` | Out decodes the data from base64. | Yes |
| `ChaChaPolySigil.In` | `func (*ChaChaPolySigil) In(data []byte) ([]byte, error)` | `github.com/dappcore/go-io/go/sigil` | In encrypts the data with pre-obfuscation. | Yes |
| `ChaChaPolySigil.Out` | `func (*ChaChaPolySigil) Out(data []byte) ([]byte, error)` | `github.com/dappcore/go-io/go/sigil` | Out decrypts the data and reverses obfuscation. | Yes |
| `GzipSigil.In` | `func (*GzipSigil) In(data []byte) ([]byte, error)` | `github.com/dappcore/go-io/go/sigil` | In compresses the data using gzip. | Yes |
| `GzipSigil.Out` | `func (*GzipSigil) Out(data []byte) ([]byte, error)` | `github.com/dappcore/go-io/go/sigil` | Out decompresses the data using gzip. | Yes |
| `HashSigil.In` | `func (*HashSigil) In(data []byte) ([]byte, error)` | `github.com/dappcore/go-io/go/sigil` | In hashes the data. | Yes |
| `HashSigil.Out` | `func (*HashSigil) Out(data []byte) ([]byte, error)` | `github.com/dappcore/go-io/go/sigil` | Out is a no-op for HashSigil. | Yes |
| `HexSigil.In` | `func (*HexSigil) In(data []byte) ([]byte, error)` | `github.com/dappcore/go-io/go/sigil` | In encodes the data to hexadecimal. | Yes |
| `HexSigil.Out` | `func (*HexSigil) Out(data []byte) ([]byte, error)` | `github.com/dappcore/go-io/go/sigil` | Out decodes the data from hexadecimal. | Yes |
| `JSONSigil.In` | `func (*JSONSigil) In(data []byte) ([]byte, error)` | `github.com/dappcore/go-io/go/sigil` | In compacts or indents the JSON data. | Yes |
| `JSONSigil.Out` | `func (*JSONSigil) Out(data []byte) ([]byte, error)` | `github.com/dappcore/go-io/go/sigil` | Out is a no-op for JSONSigil. | Yes |
| `PreObfuscator.Deobfuscate` | `Deobfuscate(data []byte, entropy []byte) []byte` | `github.com/dappcore/go-io/go/sigil` | Deobfuscate reverses the transformation after decryption. | Yes |
| `PreObfuscator.Obfuscate` | `Obfuscate(data []byte, entropy []byte) []byte` | `github.com/dappcore/go-io/go/sigil` | Obfuscate transforms plaintext before encryption using the provided entropy. | Yes |
| `ReverseSigil.In` | `func (*ReverseSigil) In(data []byte) ([]byte, error)` | `github.com/dappcore/go-io/go/sigil` | In reverses the bytes of the data. | Yes |
| `ReverseSigil.Out` | `func (*ReverseSigil) Out(data []byte) ([]byte, error)` | `github.com/dappcore/go-io/go/sigil` | Out reverses the bytes of the data. | Yes |
| `ShuffleMaskObfuscator.Deobfuscate` | `func (*ShuffleMaskObfuscator) Deobfuscate(data []byte, entropy []byte) []byte` | `github.com/dappcore/go-io/go/sigil` | Deobfuscate reverses the shuffle and mask operations. | Yes |
| `ShuffleMaskObfuscator.Obfuscate` | `func (*ShuffleMaskObfuscator) Obfuscate(data []byte, entropy []byte) []byte` | `github.com/dappcore/go-io/go/sigil` | Obfuscate shuffles bytes and applies a mask derived from entropy. | Yes |
| `Sigil.In` | `In(data []byte) ([]byte, error)` | `github.com/dappcore/go-io/go/sigil` | In applies the forward transformation to the data. | Yes |
| `Sigil.Out` | `Out(data []byte) ([]byte, error)` | `github.com/dappcore/go-io/go/sigil` | Out applies the reverse transformation to the data. | Yes |
| `XORObfuscator.Deobfuscate` | `func (*XORObfuscator) Deobfuscate(data []byte, entropy []byte) []byte` | `github.com/dappcore/go-io/go/sigil` | Deobfuscate reverses the XOR transformation (XOR is symmetric). | Yes |
| `XORObfuscator.Obfuscate` | `func (*XORObfuscator) Obfuscate(data []byte, entropy []byte) []byte` | `github.com/dappcore/go-io/go/sigil` | Obfuscate XORs the data with a key stream derived from the entropy. | Yes |
| `Medium` | `type Medium struct` | `github.com/dappcore/go-io/go/sqlite` | Medium is a SQLite-backed storage backend implementing the io.Medium interface. | Yes |
| `Option` | `type Option func(*Medium)` | `github.com/dappcore/go-io/go/sqlite` | Option configures a Medium. | Yes |
| `New` | `func New(dbPath string, opts ...Option) (*Medium, error)` | `github.com/dappcore/go-io/go/sqlite` | New creates a new SQLite Medium at the given database path. | Yes |
| `WithTable` | `func WithTable(table string) Option` | `github.com/dappcore/go-io/go/sqlite` | WithTable sets the table name (default: "files"). | Yes |
| `Medium.Append` | `func (*Medium) Append(p string) (goio.WriteCloser, error)` | `github.com/dappcore/go-io/go/sqlite` | Append opens the named file for appending, creating it if it doesn't exist. | Yes |
| `Medium.Close` | `func (*Medium) Close() error` | `github.com/dappcore/go-io/go/sqlite` | Close closes the underlying database connection. | Yes |
| `Medium.Create` | `func (*Medium) Create(p string) (goio.WriteCloser, error)` | `github.com/dappcore/go-io/go/sqlite` | Create creates or truncates the named file. | Yes |
| `Medium.Delete` | `func (*Medium) Delete(p string) error` | `github.com/dappcore/go-io/go/sqlite` | Delete removes a file or empty directory. | Yes |
| `Medium.DeleteAll` | `func (*Medium) DeleteAll(p string) error` | `github.com/dappcore/go-io/go/sqlite` | DeleteAll removes a file or directory and all its contents recursively. | Yes |
| `Medium.EnsureDir` | `func (*Medium) EnsureDir(p string) error` | `github.com/dappcore/go-io/go/sqlite` | EnsureDir makes sure a directory exists, creating it if necessary. | Yes |
| `Medium.Exists` | `func (*Medium) Exists(p string) bool` | `github.com/dappcore/go-io/go/sqlite` | Exists checks if a path exists (file or directory). | Yes |
| `Medium.FileGet` | `func (*Medium) FileGet(p string) (string, error)` | `github.com/dappcore/go-io/go/sqlite` | FileGet is a convenience function that reads a file from the medium. | Yes |
| `Medium.FileSet` | `func (*Medium) FileSet(p, content string) error` | `github.com/dappcore/go-io/go/sqlite` | FileSet is a convenience function that writes a file to the medium. | Yes |
| `Medium.IsDir` | `func (*Medium) IsDir(p string) bool` | `github.com/dappcore/go-io/go/sqlite` | IsDir checks if a path exists and is a directory. | Yes |
| `Medium.IsFile` | `func (*Medium) IsFile(p string) bool` | `github.com/dappcore/go-io/go/sqlite` | IsFile checks if a path exists and is a regular file. | Yes |
| `Medium.List` | `func (*Medium) List(p string) ([]fs.DirEntry, error)` | `github.com/dappcore/go-io/go/sqlite` | List returns the directory entries for the given path. | Yes |
| `Medium.Open` | `func (*Medium) Open(p string) (fs.File, error)` | `github.com/dappcore/go-io/go/sqlite` | Open opens the named file for reading. | Yes |
| `Medium.Read` | `func (*Medium) Read(p string) (string, error)` | `github.com/dappcore/go-io/go/sqlite` | Read retrieves the content of a file as a string. | Yes |
| `Medium.ReadStream` | `func (*Medium) ReadStream(p string) (goio.ReadCloser, error)` | `github.com/dappcore/go-io/go/sqlite` | ReadStream returns a reader for the file content. | Yes |
| `Medium.Rename` | `func (*Medium) Rename(oldPath, newPath string) error` | `github.com/dappcore/go-io/go/sqlite` | Rename moves a file or directory from oldPath to newPath. | Yes |
| `Medium.Stat` | `func (*Medium) Stat(p string) (fs.FileInfo, error)` | `github.com/dappcore/go-io/go/sqlite` | Stat returns file information for the given path. | Yes |
| `Medium.Write` | `func (*Medium) Write(p, content string) error` | `github.com/dappcore/go-io/go/sqlite` | Write saves the given content to a file, overwriting it if it exists. | Yes |
| `Medium.WriteStream` | `func (*Medium) WriteStream(p string) (goio.WriteCloser, error)` | `github.com/dappcore/go-io/go/sqlite` | WriteStream returns a writer for the file content. | Yes |
| `Medium` | `type Medium struct` | `github.com/dappcore/go-io/go/store` | Medium wraps a Store to satisfy the io.Medium interface. | Yes |
| `Store` | `type Store struct` | `github.com/dappcore/go-io/go/store` | Store is a group-namespaced key-value store backed by SQLite. | Yes |
| `New` | `func New(dbPath string) (*Store, error)` | `github.com/dappcore/go-io/go/store` | New creates a Store at the given SQLite path. | Yes |
| `NewMedium` | `func NewMedium(dbPath string) (*Medium, error)` | `github.com/dappcore/go-io/go/store` | NewMedium creates an io.Medium backed by a KV store at the given SQLite path. | Yes |
| `Medium.Append` | `func (*Medium) Append(p string) (goio.WriteCloser, error)` | `github.com/dappcore/go-io/go/store` | Append opens a key for appending. | Yes |
| `Medium.Close` | `func (*Medium) Close() error` | `github.com/dappcore/go-io/go/store` | Close closes the underlying store. | Yes |
| `Medium.Create` | `func (*Medium) Create(p string) (goio.WriteCloser, error)` | `github.com/dappcore/go-io/go/store` | Create creates or truncates a key. | Yes |
| `Medium.Delete` | `func (*Medium) Delete(p string) error` | `github.com/dappcore/go-io/go/store` | Delete removes a key, or checks that a group is empty. | Yes |
| `Medium.DeleteAll` | `func (*Medium) DeleteAll(p string) error` | `github.com/dappcore/go-io/go/store` | DeleteAll removes a key, or all keys in a group. | Yes |
| `Medium.EnsureDir` | `func (*Medium) EnsureDir(_ string) error` | `github.com/dappcore/go-io/go/store` | EnsureDir is a no-op — groups are created implicitly on Set. | No |
| `Medium.Exists` | `func (*Medium) Exists(p string) bool` | `github.com/dappcore/go-io/go/store` | Exists returns true if a group or key exists. | Yes |
| `Medium.FileGet` | `func (*Medium) FileGet(p string) (string, error)` | `github.com/dappcore/go-io/go/store` | FileGet is an alias for Read. | No |
| `Medium.FileSet` | `func (*Medium) FileSet(p, content string) error` | `github.com/dappcore/go-io/go/store` | FileSet is an alias for Write. | No |
| `Medium.IsDir` | `func (*Medium) IsDir(p string) bool` | `github.com/dappcore/go-io/go/store` | IsDir returns true if the path is a group with entries. | Yes |
| `Medium.IsFile` | `func (*Medium) IsFile(p string) bool` | `github.com/dappcore/go-io/go/store` | IsFile returns true if a group/key pair exists. | Yes |
| `Medium.List` | `func (*Medium) List(p string) ([]fs.DirEntry, error)` | `github.com/dappcore/go-io/go/store` | List returns directory entries. | Yes |
| `Medium.Open` | `func (*Medium) Open(p string) (fs.File, error)` | `github.com/dappcore/go-io/go/store` | Open opens a key for reading. | Yes |
| `Medium.Read` | `func (*Medium) Read(p string) (string, error)` | `github.com/dappcore/go-io/go/store` | Read retrieves the value at group/key. | Yes |
| `Medium.ReadStream` | `func (*Medium) ReadStream(p string) (goio.ReadCloser, error)` | `github.com/dappcore/go-io/go/store` | ReadStream returns a reader for the value. | No |
| `Medium.Rename` | `func (*Medium) Rename(oldPath, newPath string) error` | `github.com/dappcore/go-io/go/store` | Rename moves a key from one path to another. | Yes |
| `Medium.Stat` | `func (*Medium) Stat(p string) (fs.FileInfo, error)` | `github.com/dappcore/go-io/go/store` | Stat returns file info for a group (dir) or key (file). | Yes |
| `Medium.Store` | `func (*Medium) Store() *Store` | `github.com/dappcore/go-io/go/store` | Store returns the underlying KV store for direct access. | No |
| `Medium.Write` | `func (*Medium) Write(p, content string) error` | `github.com/dappcore/go-io/go/store` | Write stores a value at group/key. | Yes |
| `Medium.WriteStream` | `func (*Medium) WriteStream(p string) (goio.WriteCloser, error)` | `github.com/dappcore/go-io/go/store` | WriteStream returns a writer. | No |
| `Store.AsMedium` | `func (*Store) AsMedium() *Medium` | `github.com/dappcore/go-io/go/store` | AsMedium returns a Medium adapter for an existing Store. | Yes |
| `Store.Close` | `func (*Store) Close() error` | `github.com/dappcore/go-io/go/store` | Close closes the underlying database. | Yes |
| `Store.Count` | `func (*Store) Count(group string) (int, error)` | `github.com/dappcore/go-io/go/store` | Count returns the number of keys in a group. | Yes |
| `Store.Delete` | `func (*Store) Delete(group, key string) error` | `github.com/dappcore/go-io/go/store` | Delete removes a single key from a group. | Yes |
| `Store.DeleteGroup` | `func (*Store) DeleteGroup(group string) error` | `github.com/dappcore/go-io/go/store` | DeleteGroup removes all keys in a group. | Yes |
| `Store.Get` | `func (*Store) Get(group, key string) (string, error)` | `github.com/dappcore/go-io/go/store` | Get retrieves a value by group and key. | Yes |
| `Store.GetAll` | `func (*Store) GetAll(group string) (map[string]string, error)` | `github.com/dappcore/go-io/go/store` | GetAll returns all key-value pairs in a group. | Yes |
| `Store.Render` | `func (*Store) Render(tmplStr, group string) (string, error)` | `github.com/dappcore/go-io/go/store` | Render loads all key-value pairs from a group and renders a Go template. | Yes |
| `Store.Set` | `func (*Store) Set(group, key, value string) error` | `github.com/dappcore/go-io/go/store` | Set stores a value by group and key, overwriting if exists. | Yes |
| `Service` | `type Service struct` | `github.com/dappcore/go-io/go/workspace` | Service implements the Workspace interface. | Yes |
| `Workspace` | `type Workspace interface` | `github.com/dappcore/go-io/go/workspace` | Workspace provides management for encrypted user workspaces. | No |
| `New` | `func New(c *core.Core, crypt ...cryptProvider) (any, error)` | `github.com/dappcore/go-io/go/workspace` | New creates a new Workspace service instance. | Yes |
| `Service.CreateWorkspace` | `func (*Service) CreateWorkspace(identifier, password string) (string, error)` | `github.com/dappcore/go-io/go/workspace` | CreateWorkspace creates a new encrypted workspace. | Yes |
| `Service.HandleIPCEvents` | `func (*Service) HandleIPCEvents(c *core.Core, msg core.Message) core.Result` | `github.com/dappcore/go-io/go/workspace` | HandleIPCEvents handles workspace-related IPC messages. | No |
| `Service.SwitchWorkspace` | `func (*Service) SwitchWorkspace(name string) error` | `github.com/dappcore/go-io/go/workspace` | SwitchWorkspace changes the active workspace. | Yes |
| `Service.WorkspaceFileGet` | `func (*Service) WorkspaceFileGet(filename string) (string, error)` | `github.com/dappcore/go-io/go/workspace` | WorkspaceFileGet retrieves the content of a file from the active workspace. | Yes |
| `Service.WorkspaceFileSet` | `func (*Service) WorkspaceFileSet(filename, content string) error` | `github.com/dappcore/go-io/go/workspace` | WorkspaceFileSet saves content to a file in the active workspace. | Yes |
| `Workspace.CreateWorkspace` | `CreateWorkspace(identifier, password string) (string, error)` | `github.com/dappcore/go-io/go/workspace` | Creates a new encrypted workspace and returns its ID. | Yes |
| `Workspace.SwitchWorkspace` | `SwitchWorkspace(name string) error` | `github.com/dappcore/go-io/go/workspace` | Switches the active workspace. | Yes |
| `Workspace.WorkspaceFileGet` | `WorkspaceFileGet(filename string) (string, error)` | `github.com/dappcore/go-io/go/workspace` | Reads a file from the active workspace. | Yes |
| `Workspace.WorkspaceFileSet` | `WorkspaceFileSet(filename, content string) error` | `github.com/dappcore/go-io/go/workspace` | Writes a file into the active workspace. | Yes |
