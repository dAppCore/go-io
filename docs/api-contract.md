# API Contract

Descriptions use doc comments when present; otherwise they are short code-based summaries.
Test coverage is `Yes` when same-package tests directly execute or reference the exported symbol; otherwise `No`.
`CODEX.md` was not present in the repository at generation time.

| Name | Signature | Package Path | Description | Test Coverage |
| --- | --- | --- | --- | --- |
| `DirEntry` | `type DirEntry struct` | `dappco.re/go/core/io` | DirEntry provides a simple implementation of fs.DirEntry for mock testing. | Yes |
| `FileInfo` | `type FileInfo struct` | `dappco.re/go/core/io` | FileInfo provides a simple implementation of fs.FileInfo for mock testing. | Yes |
| `Medium` | `type Medium interface` | `dappco.re/go/core/io` | Medium defines the standard interface for a storage backend. | Yes |
| `MockFile` | `type MockFile struct` | `dappco.re/go/core/io` | MockFile implements fs.File for MockMedium. | No |
| `MockMedium` | `type MockMedium struct` | `dappco.re/go/core/io` | MockMedium is an in-memory implementation of Medium for testing. | Yes |
| `MockWriteCloser` | `type MockWriteCloser struct` | `dappco.re/go/core/io` | MockWriteCloser implements WriteCloser for MockMedium. | No |
| `Copy` | `func Copy(src Medium, srcPath string, dst Medium, dstPath string) error` | `dappco.re/go/core/io` | Copy copies a file from one medium to another. | Yes |
| `EnsureDir` | `func EnsureDir(m Medium, path string) error` | `dappco.re/go/core/io` | EnsureDir makes sure a directory exists in the given medium. | Yes |
| `IsFile` | `func IsFile(m Medium, path string) bool` | `dappco.re/go/core/io` | IsFile checks if a path exists and is a regular file in the given medium. | Yes |
| `NewMockMedium` | `func NewMockMedium() *MockMedium` | `dappco.re/go/core/io` | NewMockMedium creates a new MockMedium instance. | Yes |
| `NewSandboxed` | `func NewSandboxed(root string) (Medium, error)` | `dappco.re/go/core/io` | NewSandboxed creates a new Medium sandboxed to the given root directory. | No |
| `Read` | `func Read(m Medium, path string) (string, error)` | `dappco.re/go/core/io` | Read retrieves the content of a file from the given medium. | Yes |
| `ReadStream` | `func ReadStream(m Medium, path string) (goio.ReadCloser, error)` | `dappco.re/go/core/io` | ReadStream returns a reader for the file content from the given medium. | No |
| `Write` | `func Write(m Medium, path, content string) error` | `dappco.re/go/core/io` | Write saves the given content to a file in the given medium. | Yes |
| `WriteStream` | `func WriteStream(m Medium, path string) (goio.WriteCloser, error)` | `dappco.re/go/core/io` | WriteStream returns a writer for the file content in the given medium. | No |
| `DirEntry.Info` | `func (DirEntry) Info() (fs.FileInfo, error)` | `dappco.re/go/core/io` | Returns file info for the entry. | No |
| `DirEntry.IsDir` | `func (DirEntry) IsDir() bool` | `dappco.re/go/core/io` | Reports whether the entry represents a directory. | No |
| `DirEntry.Name` | `func (DirEntry) Name() string` | `dappco.re/go/core/io` | Returns the stored entry name. | Yes |
| `DirEntry.Type` | `func (DirEntry) Type() fs.FileMode` | `dappco.re/go/core/io` | Returns the entry type bits. | No |
| `FileInfo.IsDir` | `func (FileInfo) IsDir() bool` | `dappco.re/go/core/io` | Reports whether the entry represents a directory. | Yes |
| `FileInfo.ModTime` | `func (FileInfo) ModTime() time.Time` | `dappco.re/go/core/io` | Returns the stored modification time. | No |
| `FileInfo.Mode` | `func (FileInfo) Mode() fs.FileMode` | `dappco.re/go/core/io` | Returns the stored file mode. | No |
| `FileInfo.Name` | `func (FileInfo) Name() string` | `dappco.re/go/core/io` | Returns the stored entry name. | Yes |
| `FileInfo.Size` | `func (FileInfo) Size() int64` | `dappco.re/go/core/io` | Returns the stored size in bytes. | Yes |
| `FileInfo.Sys` | `func (FileInfo) Sys() any` | `dappco.re/go/core/io` | Returns the underlying system-specific data. | No |
| `Medium.Append` | `Append(path string) (goio.WriteCloser, error)` | `dappco.re/go/core/io` | Append opens the named file for appending, creating it if it doesn't exist. | No |
| `Medium.Create` | `Create(path string) (goio.WriteCloser, error)` | `dappco.re/go/core/io` | Create creates or truncates the named file. | No |
| `Medium.Delete` | `Delete(path string) error` | `dappco.re/go/core/io` | Delete removes a file or empty directory. | Yes |
| `Medium.DeleteAll` | `DeleteAll(path string) error` | `dappco.re/go/core/io` | DeleteAll removes a file or directory and all its contents recursively. | Yes |
| `Medium.EnsureDir` | `EnsureDir(path string) error` | `dappco.re/go/core/io` | EnsureDir makes sure a directory exists, creating it if necessary. | Yes |
| `Medium.Exists` | `Exists(path string) bool` | `dappco.re/go/core/io` | Exists checks if a path exists (file or directory). | Yes |
| `Medium.FileGet` | `FileGet(path string) (string, error)` | `dappco.re/go/core/io` | FileGet is a convenience function that reads a file from the medium. | Yes |
| `Medium.FileSet` | `FileSet(path, content string) error` | `dappco.re/go/core/io` | FileSet is a convenience function that writes a file to the medium. | Yes |
| `Medium.IsDir` | `IsDir(path string) bool` | `dappco.re/go/core/io` | IsDir checks if a path exists and is a directory. | Yes |
| `Medium.IsFile` | `IsFile(path string) bool` | `dappco.re/go/core/io` | IsFile checks if a path exists and is a regular file. | Yes |
| `Medium.List` | `List(path string) ([]fs.DirEntry, error)` | `dappco.re/go/core/io` | List returns the directory entries for the given path. | Yes |
| `Medium.Open` | `Open(path string) (fs.File, error)` | `dappco.re/go/core/io` | Open opens the named file for reading. | No |
| `Medium.Read` | `Read(path string) (string, error)` | `dappco.re/go/core/io` | Read retrieves the content of a file as a string. | Yes |
| `Medium.ReadStream` | `ReadStream(path string) (goio.ReadCloser, error)` | `dappco.re/go/core/io` | ReadStream returns a reader for the file content. | No |
| `Medium.Rename` | `Rename(oldPath, newPath string) error` | `dappco.re/go/core/io` | Rename moves a file or directory from oldPath to newPath. | Yes |
| `Medium.Stat` | `Stat(path string) (fs.FileInfo, error)` | `dappco.re/go/core/io` | Stat returns file information for the given path. | Yes |
| `Medium.Write` | `Write(path, content string) error` | `dappco.re/go/core/io` | Write saves the given content to a file, overwriting it if it exists. | Yes |
| `Medium.WriteMode` | `WriteMode(path, content string, mode os.FileMode) error` | `dappco.re/go/core/io` | WriteMode saves content with explicit file permissions. | No |
| `Medium.WriteStream` | `WriteStream(path string) (goio.WriteCloser, error)` | `dappco.re/go/core/io` | WriteStream returns a writer for the file content. | No |
| `MockFile.Close` | `func (*MockFile) Close() error` | `dappco.re/go/core/io` | Closes the current value. | No |
| `MockFile.Read` | `func (*MockFile) Read(b []byte) (int, error)` | `dappco.re/go/core/io` | Reads data from the current value. | No |
| `MockFile.Stat` | `func (*MockFile) Stat() (fs.FileInfo, error)` | `dappco.re/go/core/io` | Returns file metadata for the current value. | No |
| `MockMedium.Append` | `func (*MockMedium) Append(path string) (goio.WriteCloser, error)` | `dappco.re/go/core/io` | Append opens a file for appending in the mock filesystem. | No |
| `MockMedium.Create` | `func (*MockMedium) Create(path string) (goio.WriteCloser, error)` | `dappco.re/go/core/io` | Create creates a file in the mock filesystem. | No |
| `MockMedium.Delete` | `func (*MockMedium) Delete(path string) error` | `dappco.re/go/core/io` | Delete removes a file or empty directory from the mock filesystem. | Yes |
| `MockMedium.DeleteAll` | `func (*MockMedium) DeleteAll(path string) error` | `dappco.re/go/core/io` | DeleteAll removes a file or directory and all contents from the mock filesystem. | Yes |
| `MockMedium.EnsureDir` | `func (*MockMedium) EnsureDir(path string) error` | `dappco.re/go/core/io` | EnsureDir records that a directory exists in the mock filesystem. | Yes |
| `MockMedium.Exists` | `func (*MockMedium) Exists(path string) bool` | `dappco.re/go/core/io` | Exists checks if a path exists in the mock filesystem. | Yes |
| `MockMedium.FileGet` | `func (*MockMedium) FileGet(path string) (string, error)` | `dappco.re/go/core/io` | FileGet is a convenience function that reads a file from the mock filesystem. | Yes |
| `MockMedium.FileSet` | `func (*MockMedium) FileSet(path, content string) error` | `dappco.re/go/core/io` | FileSet is a convenience function that writes a file to the mock filesystem. | Yes |
| `MockMedium.IsDir` | `func (*MockMedium) IsDir(path string) bool` | `dappco.re/go/core/io` | IsDir checks if a path is a directory in the mock filesystem. | Yes |
| `MockMedium.IsFile` | `func (*MockMedium) IsFile(path string) bool` | `dappco.re/go/core/io` | IsFile checks if a path exists as a file in the mock filesystem. | Yes |
| `MockMedium.List` | `func (*MockMedium) List(path string) ([]fs.DirEntry, error)` | `dappco.re/go/core/io` | List returns directory entries for the mock filesystem. | Yes |
| `MockMedium.Open` | `func (*MockMedium) Open(path string) (fs.File, error)` | `dappco.re/go/core/io` | Open opens a file from the mock filesystem. | No |
| `MockMedium.Read` | `func (*MockMedium) Read(path string) (string, error)` | `dappco.re/go/core/io` | Read retrieves the content of a file from the mock filesystem. | Yes |
| `MockMedium.ReadStream` | `func (*MockMedium) ReadStream(path string) (goio.ReadCloser, error)` | `dappco.re/go/core/io` | ReadStream returns a reader for the file content in the mock filesystem. | No |
| `MockMedium.Rename` | `func (*MockMedium) Rename(oldPath, newPath string) error` | `dappco.re/go/core/io` | Rename moves a file or directory in the mock filesystem. | Yes |
| `MockMedium.Stat` | `func (*MockMedium) Stat(path string) (fs.FileInfo, error)` | `dappco.re/go/core/io` | Stat returns file information for the mock filesystem. | Yes |
| `MockMedium.Write` | `func (*MockMedium) Write(path, content string) error` | `dappco.re/go/core/io` | Write saves the given content to a file in the mock filesystem. | Yes |
| `MockMedium.WriteMode` | `func (*MockMedium) WriteMode(path, content string, mode os.FileMode) error` | `dappco.re/go/core/io` | Writes content using an explicit file mode. | No |
| `MockMedium.WriteStream` | `func (*MockMedium) WriteStream(path string) (goio.WriteCloser, error)` | `dappco.re/go/core/io` | WriteStream returns a writer for the file content in the mock filesystem. | No |
| `MockWriteCloser.Close` | `func (*MockWriteCloser) Close() error` | `dappco.re/go/core/io` | Closes the current value. | No |
| `MockWriteCloser.Write` | `func (*MockWriteCloser) Write(p []byte) (int, error)` | `dappco.re/go/core/io` | Writes data to the current value. | No |
| `Medium` | `type Medium struct` | `dappco.re/go/core/io/datanode` | Medium is an in-memory storage backend backed by a Borg DataNode. | Yes |
| `FromTar` | `func FromTar(data []byte) (*Medium, error)` | `dappco.re/go/core/io/datanode` | FromTar creates a Medium from a tarball, restoring all files. | Yes |
| `New` | `func New() *Medium` | `dappco.re/go/core/io/datanode` | New creates a new empty DataNode Medium. | Yes |
| `Medium.Append` | `func (*Medium) Append(p string) (goio.WriteCloser, error)` | `dappco.re/go/core/io/datanode` | Opens the named file for appending, creating it if needed. | Yes |
| `Medium.Create` | `func (*Medium) Create(p string) (goio.WriteCloser, error)` | `dappco.re/go/core/io/datanode` | Creates or truncates the named file and returns a writer. | Yes |
| `Medium.DataNode` | `func (*Medium) DataNode() *datanode.DataNode` | `dappco.re/go/core/io/datanode` | DataNode returns the underlying Borg DataNode. | Yes |
| `Medium.Delete` | `func (*Medium) Delete(p string) error` | `dappco.re/go/core/io/datanode` | Removes a file, key, or empty directory. | Yes |
| `Medium.DeleteAll` | `func (*Medium) DeleteAll(p string) error` | `dappco.re/go/core/io/datanode` | Removes a file or directory tree recursively. | Yes |
| `Medium.EnsureDir` | `func (*Medium) EnsureDir(p string) error` | `dappco.re/go/core/io/datanode` | Ensures a directory path exists. | Yes |
| `Medium.Exists` | `func (*Medium) Exists(p string) bool` | `dappco.re/go/core/io/datanode` | Reports whether the path exists. | Yes |
| `Medium.FileGet` | `func (*Medium) FileGet(p string) (string, error)` | `dappco.re/go/core/io/datanode` | Reads a file or key through the convenience accessor. | Yes |
| `Medium.FileSet` | `func (*Medium) FileSet(p, content string) error` | `dappco.re/go/core/io/datanode` | Writes a file or key through the convenience accessor. | Yes |
| `Medium.IsDir` | `func (*Medium) IsDir(p string) bool` | `dappco.re/go/core/io/datanode` | Reports whether the entry represents a directory. | Yes |
| `Medium.IsFile` | `func (*Medium) IsFile(p string) bool` | `dappco.re/go/core/io/datanode` | Reports whether the path exists as a regular file. | Yes |
| `Medium.List` | `func (*Medium) List(p string) ([]fs.DirEntry, error)` | `dappco.re/go/core/io/datanode` | Lists directory entries beneath the given path. | Yes |
| `Medium.Open` | `func (*Medium) Open(p string) (fs.File, error)` | `dappco.re/go/core/io/datanode` | Opens the named file for reading. | Yes |
| `Medium.Read` | `func (*Medium) Read(p string) (string, error)` | `dappco.re/go/core/io/datanode` | Reads data from the current value. | Yes |
| `Medium.ReadStream` | `func (*Medium) ReadStream(p string) (goio.ReadCloser, error)` | `dappco.re/go/core/io/datanode` | Opens a streaming reader for the file content. | Yes |
| `Medium.Rename` | `func (*Medium) Rename(oldPath, newPath string) error` | `dappco.re/go/core/io/datanode` | Moves a file or directory to a new path. | Yes |
| `Medium.Restore` | `func (*Medium) Restore(data []byte) error` | `dappco.re/go/core/io/datanode` | Restore replaces the filesystem contents from a tarball. | Yes |
| `Medium.Snapshot` | `func (*Medium) Snapshot() ([]byte, error)` | `dappco.re/go/core/io/datanode` | Snapshot serializes the entire filesystem to a tarball. | Yes |
| `Medium.Stat` | `func (*Medium) Stat(p string) (fs.FileInfo, error)` | `dappco.re/go/core/io/datanode` | Returns file metadata for the current value. | Yes |
| `Medium.Write` | `func (*Medium) Write(p, content string) error` | `dappco.re/go/core/io/datanode` | Writes data to the current value. | Yes |
| `Medium.WriteMode` | `func (*Medium) WriteMode(p, content string, mode os.FileMode) error` | `dappco.re/go/core/io/datanode` | Writes content using an explicit file mode. | No |
| `Medium.WriteStream` | `func (*Medium) WriteStream(p string) (goio.WriteCloser, error)` | `dappco.re/go/core/io/datanode` | Opens a streaming writer for the file content. | Yes |
| `Medium` | `type Medium struct` | `dappco.re/go/core/io/local` | Medium is a local filesystem storage backend. | Yes |
| `New` | `func New(root string) (*Medium, error)` | `dappco.re/go/core/io/local` | New creates a new local Medium rooted at the given directory. | Yes |
| `Medium.Append` | `func (*Medium) Append(p string) (goio.WriteCloser, error)` | `dappco.re/go/core/io/local` | Append opens the named file for appending, creating it if it doesn't exist. | No |
| `Medium.Create` | `func (*Medium) Create(p string) (goio.WriteCloser, error)` | `dappco.re/go/core/io/local` | Create creates or truncates the named file. | Yes |
| `Medium.Delete` | `func (*Medium) Delete(p string) error` | `dappco.re/go/core/io/local` | Delete removes a file or empty directory. | Yes |
| `Medium.DeleteAll` | `func (*Medium) DeleteAll(p string) error` | `dappco.re/go/core/io/local` | DeleteAll removes a file or directory recursively. | Yes |
| `Medium.EnsureDir` | `func (*Medium) EnsureDir(p string) error` | `dappco.re/go/core/io/local` | EnsureDir creates directory if it doesn't exist. | Yes |
| `Medium.Exists` | `func (*Medium) Exists(p string) bool` | `dappco.re/go/core/io/local` | Exists returns true if path exists. | Yes |
| `Medium.FileGet` | `func (*Medium) FileGet(p string) (string, error)` | `dappco.re/go/core/io/local` | FileGet is an alias for Read. | Yes |
| `Medium.FileSet` | `func (*Medium) FileSet(p, content string) error` | `dappco.re/go/core/io/local` | FileSet is an alias for Write. | Yes |
| `Medium.IsDir` | `func (*Medium) IsDir(p string) bool` | `dappco.re/go/core/io/local` | IsDir returns true if path is a directory. | Yes |
| `Medium.IsFile` | `func (*Medium) IsFile(p string) bool` | `dappco.re/go/core/io/local` | IsFile returns true if path is a regular file. | Yes |
| `Medium.List` | `func (*Medium) List(p string) ([]fs.DirEntry, error)` | `dappco.re/go/core/io/local` | List returns directory entries. | Yes |
| `Medium.Open` | `func (*Medium) Open(p string) (fs.File, error)` | `dappco.re/go/core/io/local` | Open opens the named file for reading. | Yes |
| `Medium.Read` | `func (*Medium) Read(p string) (string, error)` | `dappco.re/go/core/io/local` | Read returns file contents as string. | Yes |
| `Medium.ReadStream` | `func (*Medium) ReadStream(path string) (goio.ReadCloser, error)` | `dappco.re/go/core/io/local` | ReadStream returns a reader for the file content. | Yes |
| `Medium.Rename` | `func (*Medium) Rename(oldPath, newPath string) error` | `dappco.re/go/core/io/local` | Rename moves a file or directory. | Yes |
| `Medium.Stat` | `func (*Medium) Stat(p string) (fs.FileInfo, error)` | `dappco.re/go/core/io/local` | Stat returns file info. | Yes |
| `Medium.Write` | `func (*Medium) Write(p, content string) error` | `dappco.re/go/core/io/local` | Write saves content to file, creating parent directories as needed. | Yes |
| `Medium.WriteMode` | `func (*Medium) WriteMode(p, content string, mode os.FileMode) error` | `dappco.re/go/core/io/local` | WriteMode saves content to file with explicit permissions. | Yes |
| `Medium.WriteStream` | `func (*Medium) WriteStream(path string) (goio.WriteCloser, error)` | `dappco.re/go/core/io/local` | WriteStream returns a writer for the file content. | Yes |
| `Node` | `type Node struct` | `dappco.re/go/core/io/node` | Node is an in-memory filesystem that implements coreio.Node (and therefore coreio.Medium). | Yes |
| `WalkOptions` | `type WalkOptions struct` | `dappco.re/go/core/io/node` | WalkOptions configures the behaviour of Walk. | Yes |
| `FromTar` | `func FromTar(data []byte) (*Node, error)` | `dappco.re/go/core/io/node` | FromTar creates a new Node from a tar archive. | Yes |
| `New` | `func New() *Node` | `dappco.re/go/core/io/node` | New creates a new, empty Node. | Yes |
| `Node.AddData` | `func (*Node) AddData(name string, content []byte)` | `dappco.re/go/core/io/node` | AddData stages content in the in-memory filesystem. | Yes |
| `Node.Append` | `func (*Node) Append(p string) (goio.WriteCloser, error)` | `dappco.re/go/core/io/node` | Append opens the named file for appending, creating it if needed. | No |
| `Node.CopyFile` | `func (*Node) CopyFile(src, dst string, perm fs.FileMode) error` | `dappco.re/go/core/io/node` | CopyFile copies a file from the in-memory tree to the local filesystem. | Yes |
| `Node.CopyTo` | `func (*Node) CopyTo(target coreio.Medium, sourcePath, destPath string) error` | `dappco.re/go/core/io/node` | CopyTo copies a file (or directory tree) from the node to any Medium. | No |
| `Node.Create` | `func (*Node) Create(p string) (goio.WriteCloser, error)` | `dappco.re/go/core/io/node` | Create creates or truncates the named file, returning a WriteCloser. | No |
| `Node.Delete` | `func (*Node) Delete(p string) error` | `dappco.re/go/core/io/node` | Delete removes a single file. | No |
| `Node.DeleteAll` | `func (*Node) DeleteAll(p string) error` | `dappco.re/go/core/io/node` | DeleteAll removes a file or directory and all children. | No |
| `Node.EnsureDir` | `func (*Node) EnsureDir(_ string) error` | `dappco.re/go/core/io/node` | EnsureDir is a no-op because directories are implicit in Node. | No |
| `Node.Exists` | `func (*Node) Exists(p string) bool` | `dappco.re/go/core/io/node` | Exists checks if a path exists (file or directory). | Yes |
| `Node.FileGet` | `func (*Node) FileGet(p string) (string, error)` | `dappco.re/go/core/io/node` | FileGet is an alias for Read. | No |
| `Node.FileSet` | `func (*Node) FileSet(p, content string) error` | `dappco.re/go/core/io/node` | FileSet is an alias for Write. | No |
| `Node.IsDir` | `func (*Node) IsDir(p string) bool` | `dappco.re/go/core/io/node` | IsDir checks if a path exists and is a directory. | No |
| `Node.IsFile` | `func (*Node) IsFile(p string) bool` | `dappco.re/go/core/io/node` | IsFile checks if a path exists and is a regular file. | No |
| `Node.List` | `func (*Node) List(p string) ([]fs.DirEntry, error)` | `dappco.re/go/core/io/node` | List returns directory entries for the given path. | No |
| `Node.LoadTar` | `func (*Node) LoadTar(data []byte) error` | `dappco.re/go/core/io/node` | LoadTar replaces the in-memory tree with the contents of a tar archive. | Yes |
| `Node.Open` | `func (*Node) Open(name string) (fs.File, error)` | `dappco.re/go/core/io/node` | Open opens a file from the Node. | Yes |
| `Node.Read` | `func (*Node) Read(p string) (string, error)` | `dappco.re/go/core/io/node` | Read retrieves the content of a file as a string. | No |
| `Node.ReadDir` | `func (*Node) ReadDir(name string) ([]fs.DirEntry, error)` | `dappco.re/go/core/io/node` | ReadDir reads and returns all directory entries for the named directory. | Yes |
| `Node.ReadFile` | `func (*Node) ReadFile(name string) ([]byte, error)` | `dappco.re/go/core/io/node` | ReadFile returns the content of the named file as a byte slice. | Yes |
| `Node.ReadStream` | `func (*Node) ReadStream(p string) (goio.ReadCloser, error)` | `dappco.re/go/core/io/node` | ReadStream returns a ReadCloser for the file content. | No |
| `Node.Rename` | `func (*Node) Rename(oldPath, newPath string) error` | `dappco.re/go/core/io/node` | Rename moves a file from oldPath to newPath. | No |
| `Node.Stat` | `func (*Node) Stat(name string) (fs.FileInfo, error)` | `dappco.re/go/core/io/node` | Stat returns file information for the given path. | Yes |
| `Node.ToTar` | `func (*Node) ToTar() ([]byte, error)` | `dappco.re/go/core/io/node` | ToTar serialises the entire in-memory tree to a tar archive. | Yes |
| `Node.Walk` | `func (*Node) Walk(root string, fn fs.WalkDirFunc, opts ...WalkOptions) error` | `dappco.re/go/core/io/node` | Walk walks the in-memory tree with optional WalkOptions. | Yes |
| `Node.WalkNode` | `func (*Node) WalkNode(root string, fn fs.WalkDirFunc) error` | `dappco.re/go/core/io/node` | WalkNode walks the in-memory tree, calling fn for each entry. | No |
| `Node.Write` | `func (*Node) Write(p, content string) error` | `dappco.re/go/core/io/node` | Write saves the given content to a file, overwriting it if it exists. | No |
| `Node.WriteMode` | `func (*Node) WriteMode(p, content string, mode os.FileMode) error` | `dappco.re/go/core/io/node` | WriteMode saves content with explicit permissions (no-op for in-memory node). | No |
| `Node.WriteStream` | `func (*Node) WriteStream(p string) (goio.WriteCloser, error)` | `dappco.re/go/core/io/node` | WriteStream returns a WriteCloser for the file content. | No |
| `Medium` | `type Medium struct` | `dappco.re/go/core/io/s3` | Medium is an S3-backed storage backend implementing the io.Medium interface. | Yes |
| `Option` | `type Option func(*Medium)` | `dappco.re/go/core/io/s3` | Option configures a Medium. | Yes |
| `New` | `func New(bucket string, opts ...Option) (*Medium, error)` | `dappco.re/go/core/io/s3` | New creates a new S3 Medium for the given bucket. | Yes |
| `WithClient` | `func WithClient(client *s3.Client) Option` | `dappco.re/go/core/io/s3` | WithClient sets the S3 client for dependency injection. | No |
| `WithPrefix` | `func WithPrefix(prefix string) Option` | `dappco.re/go/core/io/s3` | WithPrefix sets an optional key prefix for all operations. | Yes |
| `Medium.Append` | `func (*Medium) Append(p string) (goio.WriteCloser, error)` | `dappco.re/go/core/io/s3` | Append opens the named file for appending. | Yes |
| `Medium.Create` | `func (*Medium) Create(p string) (goio.WriteCloser, error)` | `dappco.re/go/core/io/s3` | Create creates or truncates the named file. | Yes |
| `Medium.Delete` | `func (*Medium) Delete(p string) error` | `dappco.re/go/core/io/s3` | Delete removes a single object. | Yes |
| `Medium.DeleteAll` | `func (*Medium) DeleteAll(p string) error` | `dappco.re/go/core/io/s3` | DeleteAll removes all objects under the given prefix. | Yes |
| `Medium.EnsureDir` | `func (*Medium) EnsureDir(_ string) error` | `dappco.re/go/core/io/s3` | EnsureDir is a no-op for S3 (S3 has no real directories). | Yes |
| `Medium.Exists` | `func (*Medium) Exists(p string) bool` | `dappco.re/go/core/io/s3` | Exists checks if a path exists (file or directory prefix). | Yes |
| `Medium.FileGet` | `func (*Medium) FileGet(p string) (string, error)` | `dappco.re/go/core/io/s3` | FileGet is a convenience function that reads a file from the medium. | Yes |
| `Medium.FileSet` | `func (*Medium) FileSet(p, content string) error` | `dappco.re/go/core/io/s3` | FileSet is a convenience function that writes a file to the medium. | Yes |
| `Medium.IsDir` | `func (*Medium) IsDir(p string) bool` | `dappco.re/go/core/io/s3` | IsDir checks if a path exists and is a directory (has objects under it as a prefix). | Yes |
| `Medium.IsFile` | `func (*Medium) IsFile(p string) bool` | `dappco.re/go/core/io/s3` | IsFile checks if a path exists and is a regular file (not a "directory" prefix). | Yes |
| `Medium.List` | `func (*Medium) List(p string) ([]fs.DirEntry, error)` | `dappco.re/go/core/io/s3` | List returns directory entries for the given path using ListObjectsV2 with delimiter. | Yes |
| `Medium.Open` | `func (*Medium) Open(p string) (fs.File, error)` | `dappco.re/go/core/io/s3` | Open opens the named file for reading. | Yes |
| `Medium.Read` | `func (*Medium) Read(p string) (string, error)` | `dappco.re/go/core/io/s3` | Read retrieves the content of a file as a string. | Yes |
| `Medium.ReadStream` | `func (*Medium) ReadStream(p string) (goio.ReadCloser, error)` | `dappco.re/go/core/io/s3` | ReadStream returns a reader for the file content. | Yes |
| `Medium.Rename` | `func (*Medium) Rename(oldPath, newPath string) error` | `dappco.re/go/core/io/s3` | Rename moves an object by copying then deleting the original. | Yes |
| `Medium.Stat` | `func (*Medium) Stat(p string) (fs.FileInfo, error)` | `dappco.re/go/core/io/s3` | Stat returns file information for the given path using HeadObject. | Yes |
| `Medium.Write` | `func (*Medium) Write(p, content string) error` | `dappco.re/go/core/io/s3` | Write saves the given content to a file, overwriting it if it exists. | Yes |
| `Medium.WriteStream` | `func (*Medium) WriteStream(p string) (goio.WriteCloser, error)` | `dappco.re/go/core/io/s3` | WriteStream returns a writer for the file content. | Yes |
| `Base64Sigil` | `type Base64Sigil struct` | `dappco.re/go/core/io/sigil` | Base64Sigil is a Sigil that encodes/decodes data to/from base64. | Yes |
| `ChaChaPolySigil` | `type ChaChaPolySigil struct` | `dappco.re/go/core/io/sigil` | ChaChaPolySigil is a Sigil that encrypts/decrypts data using ChaCha20-Poly1305. | Yes |
| `GzipSigil` | `type GzipSigil struct` | `dappco.re/go/core/io/sigil` | GzipSigil is a Sigil that compresses/decompresses data using gzip. | Yes |
| `HashSigil` | `type HashSigil struct` | `dappco.re/go/core/io/sigil` | HashSigil is a Sigil that hashes the data using a specified algorithm. | Yes |
| `HexSigil` | `type HexSigil struct` | `dappco.re/go/core/io/sigil` | HexSigil is a Sigil that encodes/decodes data to/from hexadecimal. | Yes |
| `JSONSigil` | `type JSONSigil struct` | `dappco.re/go/core/io/sigil` | JSONSigil is a Sigil that compacts or indents JSON data. | Yes |
| `PreObfuscator` | `type PreObfuscator interface` | `dappco.re/go/core/io/sigil` | PreObfuscator applies a reversible transformation to data before encryption. | Yes |
| `ReverseSigil` | `type ReverseSigil struct` | `dappco.re/go/core/io/sigil` | ReverseSigil is a Sigil that reverses the bytes of the payload. | Yes |
| `ShuffleMaskObfuscator` | `type ShuffleMaskObfuscator struct` | `dappco.re/go/core/io/sigil` | ShuffleMaskObfuscator provides stronger obfuscation through byte shuffling and masking. | Yes |
| `Sigil` | `type Sigil interface` | `dappco.re/go/core/io/sigil` | Sigil defines the interface for a data transformer. | Yes |
| `XORObfuscator` | `type XORObfuscator struct` | `dappco.re/go/core/io/sigil` | XORObfuscator performs XOR-based obfuscation using an entropy-derived key stream. | Yes |
| `GetNonceFromCiphertext` | `func GetNonceFromCiphertext(ciphertext []byte) ([]byte, error)` | `dappco.re/go/core/io/sigil` | GetNonceFromCiphertext extracts the nonce from encrypted output. | Yes |
| `NewChaChaPolySigil` | `func NewChaChaPolySigil(key []byte) (*ChaChaPolySigil, error)` | `dappco.re/go/core/io/sigil` | NewChaChaPolySigil creates a new encryption sigil with the given key. | Yes |
| `NewChaChaPolySigilWithObfuscator` | `func NewChaChaPolySigilWithObfuscator(key []byte, obfuscator PreObfuscator) (*ChaChaPolySigil, error)` | `dappco.re/go/core/io/sigil` | NewChaChaPolySigilWithObfuscator creates a new encryption sigil with custom obfuscator. | Yes |
| `NewHashSigil` | `func NewHashSigil(h crypto.Hash) *HashSigil` | `dappco.re/go/core/io/sigil` | NewHashSigil creates a new HashSigil. | Yes |
| `NewSigil` | `func NewSigil(name string) (Sigil, error)` | `dappco.re/go/core/io/sigil` | NewSigil is a factory function that returns a Sigil based on a string name. | Yes |
| `Transmute` | `func Transmute(data []byte, sigils []Sigil) ([]byte, error)` | `dappco.re/go/core/io/sigil` | Transmute applies a series of sigils to data in sequence. | Yes |
| `Untransmute` | `func Untransmute(data []byte, sigils []Sigil) ([]byte, error)` | `dappco.re/go/core/io/sigil` | Untransmute reverses a transmutation by applying Out in reverse order. | Yes |
| `Base64Sigil.In` | `func (*Base64Sigil) In(data []byte) ([]byte, error)` | `dappco.re/go/core/io/sigil` | In encodes the data to base64. | Yes |
| `Base64Sigil.Out` | `func (*Base64Sigil) Out(data []byte) ([]byte, error)` | `dappco.re/go/core/io/sigil` | Out decodes the data from base64. | Yes |
| `ChaChaPolySigil.In` | `func (*ChaChaPolySigil) In(data []byte) ([]byte, error)` | `dappco.re/go/core/io/sigil` | In encrypts the data with pre-obfuscation. | Yes |
| `ChaChaPolySigil.Out` | `func (*ChaChaPolySigil) Out(data []byte) ([]byte, error)` | `dappco.re/go/core/io/sigil` | Out decrypts the data and reverses obfuscation. | Yes |
| `GzipSigil.In` | `func (*GzipSigil) In(data []byte) ([]byte, error)` | `dappco.re/go/core/io/sigil` | In compresses the data using gzip. | Yes |
| `GzipSigil.Out` | `func (*GzipSigil) Out(data []byte) ([]byte, error)` | `dappco.re/go/core/io/sigil` | Out decompresses the data using gzip. | Yes |
| `HashSigil.In` | `func (*HashSigil) In(data []byte) ([]byte, error)` | `dappco.re/go/core/io/sigil` | In hashes the data. | Yes |
| `HashSigil.Out` | `func (*HashSigil) Out(data []byte) ([]byte, error)` | `dappco.re/go/core/io/sigil` | Out is a no-op for HashSigil. | Yes |
| `HexSigil.In` | `func (*HexSigil) In(data []byte) ([]byte, error)` | `dappco.re/go/core/io/sigil` | In encodes the data to hexadecimal. | Yes |
| `HexSigil.Out` | `func (*HexSigil) Out(data []byte) ([]byte, error)` | `dappco.re/go/core/io/sigil` | Out decodes the data from hexadecimal. | Yes |
| `JSONSigil.In` | `func (*JSONSigil) In(data []byte) ([]byte, error)` | `dappco.re/go/core/io/sigil` | In compacts or indents the JSON data. | Yes |
| `JSONSigil.Out` | `func (*JSONSigil) Out(data []byte) ([]byte, error)` | `dappco.re/go/core/io/sigil` | Out is a no-op for JSONSigil. | Yes |
| `PreObfuscator.Deobfuscate` | `Deobfuscate(data []byte, entropy []byte) []byte` | `dappco.re/go/core/io/sigil` | Deobfuscate reverses the transformation after decryption. | Yes |
| `PreObfuscator.Obfuscate` | `Obfuscate(data []byte, entropy []byte) []byte` | `dappco.re/go/core/io/sigil` | Obfuscate transforms plaintext before encryption using the provided entropy. | Yes |
| `ReverseSigil.In` | `func (*ReverseSigil) In(data []byte) ([]byte, error)` | `dappco.re/go/core/io/sigil` | In reverses the bytes of the data. | Yes |
| `ReverseSigil.Out` | `func (*ReverseSigil) Out(data []byte) ([]byte, error)` | `dappco.re/go/core/io/sigil` | Out reverses the bytes of the data. | Yes |
| `ShuffleMaskObfuscator.Deobfuscate` | `func (*ShuffleMaskObfuscator) Deobfuscate(data []byte, entropy []byte) []byte` | `dappco.re/go/core/io/sigil` | Deobfuscate reverses the shuffle and mask operations. | Yes |
| `ShuffleMaskObfuscator.Obfuscate` | `func (*ShuffleMaskObfuscator) Obfuscate(data []byte, entropy []byte) []byte` | `dappco.re/go/core/io/sigil` | Obfuscate shuffles bytes and applies a mask derived from entropy. | Yes |
| `Sigil.In` | `In(data []byte) ([]byte, error)` | `dappco.re/go/core/io/sigil` | In applies the forward transformation to the data. | Yes |
| `Sigil.Out` | `Out(data []byte) ([]byte, error)` | `dappco.re/go/core/io/sigil` | Out applies the reverse transformation to the data. | Yes |
| `XORObfuscator.Deobfuscate` | `func (*XORObfuscator) Deobfuscate(data []byte, entropy []byte) []byte` | `dappco.re/go/core/io/sigil` | Deobfuscate reverses the XOR transformation (XOR is symmetric). | Yes |
| `XORObfuscator.Obfuscate` | `func (*XORObfuscator) Obfuscate(data []byte, entropy []byte) []byte` | `dappco.re/go/core/io/sigil` | Obfuscate XORs the data with a key stream derived from the entropy. | Yes |
| `Medium` | `type Medium struct` | `dappco.re/go/core/io/sqlite` | Medium is a SQLite-backed storage backend implementing the io.Medium interface. | Yes |
| `Option` | `type Option func(*Medium)` | `dappco.re/go/core/io/sqlite` | Option configures a Medium. | Yes |
| `New` | `func New(dbPath string, opts ...Option) (*Medium, error)` | `dappco.re/go/core/io/sqlite` | New creates a new SQLite Medium at the given database path. | Yes |
| `WithTable` | `func WithTable(table string) Option` | `dappco.re/go/core/io/sqlite` | WithTable sets the table name (default: "files"). | Yes |
| `Medium.Append` | `func (*Medium) Append(p string) (goio.WriteCloser, error)` | `dappco.re/go/core/io/sqlite` | Append opens the named file for appending, creating it if it doesn't exist. | Yes |
| `Medium.Close` | `func (*Medium) Close() error` | `dappco.re/go/core/io/sqlite` | Close closes the underlying database connection. | Yes |
| `Medium.Create` | `func (*Medium) Create(p string) (goio.WriteCloser, error)` | `dappco.re/go/core/io/sqlite` | Create creates or truncates the named file. | Yes |
| `Medium.Delete` | `func (*Medium) Delete(p string) error` | `dappco.re/go/core/io/sqlite` | Delete removes a file or empty directory. | Yes |
| `Medium.DeleteAll` | `func (*Medium) DeleteAll(p string) error` | `dappco.re/go/core/io/sqlite` | DeleteAll removes a file or directory and all its contents recursively. | Yes |
| `Medium.EnsureDir` | `func (*Medium) EnsureDir(p string) error` | `dappco.re/go/core/io/sqlite` | EnsureDir makes sure a directory exists, creating it if necessary. | Yes |
| `Medium.Exists` | `func (*Medium) Exists(p string) bool` | `dappco.re/go/core/io/sqlite` | Exists checks if a path exists (file or directory). | Yes |
| `Medium.FileGet` | `func (*Medium) FileGet(p string) (string, error)` | `dappco.re/go/core/io/sqlite` | FileGet is a convenience function that reads a file from the medium. | Yes |
| `Medium.FileSet` | `func (*Medium) FileSet(p, content string) error` | `dappco.re/go/core/io/sqlite` | FileSet is a convenience function that writes a file to the medium. | Yes |
| `Medium.IsDir` | `func (*Medium) IsDir(p string) bool` | `dappco.re/go/core/io/sqlite` | IsDir checks if a path exists and is a directory. | Yes |
| `Medium.IsFile` | `func (*Medium) IsFile(p string) bool` | `dappco.re/go/core/io/sqlite` | IsFile checks if a path exists and is a regular file. | Yes |
| `Medium.List` | `func (*Medium) List(p string) ([]fs.DirEntry, error)` | `dappco.re/go/core/io/sqlite` | List returns the directory entries for the given path. | Yes |
| `Medium.Open` | `func (*Medium) Open(p string) (fs.File, error)` | `dappco.re/go/core/io/sqlite` | Open opens the named file for reading. | Yes |
| `Medium.Read` | `func (*Medium) Read(p string) (string, error)` | `dappco.re/go/core/io/sqlite` | Read retrieves the content of a file as a string. | Yes |
| `Medium.ReadStream` | `func (*Medium) ReadStream(p string) (goio.ReadCloser, error)` | `dappco.re/go/core/io/sqlite` | ReadStream returns a reader for the file content. | Yes |
| `Medium.Rename` | `func (*Medium) Rename(oldPath, newPath string) error` | `dappco.re/go/core/io/sqlite` | Rename moves a file or directory from oldPath to newPath. | Yes |
| `Medium.Stat` | `func (*Medium) Stat(p string) (fs.FileInfo, error)` | `dappco.re/go/core/io/sqlite` | Stat returns file information for the given path. | Yes |
| `Medium.Write` | `func (*Medium) Write(p, content string) error` | `dappco.re/go/core/io/sqlite` | Write saves the given content to a file, overwriting it if it exists. | Yes |
| `Medium.WriteStream` | `func (*Medium) WriteStream(p string) (goio.WriteCloser, error)` | `dappco.re/go/core/io/sqlite` | WriteStream returns a writer for the file content. | Yes |
| `Medium` | `type Medium struct` | `dappco.re/go/core/io/store` | Medium wraps a Store to satisfy the io.Medium interface. | Yes |
| `Store` | `type Store struct` | `dappco.re/go/core/io/store` | Store is a group-namespaced key-value store backed by SQLite. | Yes |
| `New` | `func New(dbPath string) (*Store, error)` | `dappco.re/go/core/io/store` | New creates a Store at the given SQLite path. | Yes |
| `NewMedium` | `func NewMedium(dbPath string) (*Medium, error)` | `dappco.re/go/core/io/store` | NewMedium creates an io.Medium backed by a KV store at the given SQLite path. | Yes |
| `Medium.Append` | `func (*Medium) Append(p string) (goio.WriteCloser, error)` | `dappco.re/go/core/io/store` | Append opens a key for appending. | Yes |
| `Medium.Close` | `func (*Medium) Close() error` | `dappco.re/go/core/io/store` | Close closes the underlying store. | Yes |
| `Medium.Create` | `func (*Medium) Create(p string) (goio.WriteCloser, error)` | `dappco.re/go/core/io/store` | Create creates or truncates a key. | Yes |
| `Medium.Delete` | `func (*Medium) Delete(p string) error` | `dappco.re/go/core/io/store` | Delete removes a key, or checks that a group is empty. | Yes |
| `Medium.DeleteAll` | `func (*Medium) DeleteAll(p string) error` | `dappco.re/go/core/io/store` | DeleteAll removes a key, or all keys in a group. | Yes |
| `Medium.EnsureDir` | `func (*Medium) EnsureDir(_ string) error` | `dappco.re/go/core/io/store` | EnsureDir is a no-op — groups are created implicitly on Set. | No |
| `Medium.Exists` | `func (*Medium) Exists(p string) bool` | `dappco.re/go/core/io/store` | Exists returns true if a group or key exists. | Yes |
| `Medium.FileGet` | `func (*Medium) FileGet(p string) (string, error)` | `dappco.re/go/core/io/store` | FileGet is an alias for Read. | No |
| `Medium.FileSet` | `func (*Medium) FileSet(p, content string) error` | `dappco.re/go/core/io/store` | FileSet is an alias for Write. | No |
| `Medium.IsDir` | `func (*Medium) IsDir(p string) bool` | `dappco.re/go/core/io/store` | IsDir returns true if the path is a group with entries. | Yes |
| `Medium.IsFile` | `func (*Medium) IsFile(p string) bool` | `dappco.re/go/core/io/store` | IsFile returns true if a group/key pair exists. | Yes |
| `Medium.List` | `func (*Medium) List(p string) ([]fs.DirEntry, error)` | `dappco.re/go/core/io/store` | List returns directory entries. | Yes |
| `Medium.Open` | `func (*Medium) Open(p string) (fs.File, error)` | `dappco.re/go/core/io/store` | Open opens a key for reading. | Yes |
| `Medium.Read` | `func (*Medium) Read(p string) (string, error)` | `dappco.re/go/core/io/store` | Read retrieves the value at group/key. | Yes |
| `Medium.ReadStream` | `func (*Medium) ReadStream(p string) (goio.ReadCloser, error)` | `dappco.re/go/core/io/store` | ReadStream returns a reader for the value. | No |
| `Medium.Rename` | `func (*Medium) Rename(oldPath, newPath string) error` | `dappco.re/go/core/io/store` | Rename moves a key from one path to another. | Yes |
| `Medium.Stat` | `func (*Medium) Stat(p string) (fs.FileInfo, error)` | `dappco.re/go/core/io/store` | Stat returns file info for a group (dir) or key (file). | Yes |
| `Medium.Store` | `func (*Medium) Store() *Store` | `dappco.re/go/core/io/store` | Store returns the underlying KV store for direct access. | No |
| `Medium.Write` | `func (*Medium) Write(p, content string) error` | `dappco.re/go/core/io/store` | Write stores a value at group/key. | Yes |
| `Medium.WriteStream` | `func (*Medium) WriteStream(p string) (goio.WriteCloser, error)` | `dappco.re/go/core/io/store` | WriteStream returns a writer. | No |
| `Store.AsMedium` | `func (*Store) AsMedium() *Medium` | `dappco.re/go/core/io/store` | AsMedium returns a Medium adapter for an existing Store. | Yes |
| `Store.Close` | `func (*Store) Close() error` | `dappco.re/go/core/io/store` | Close closes the underlying database. | Yes |
| `Store.Count` | `func (*Store) Count(group string) (int, error)` | `dappco.re/go/core/io/store` | Count returns the number of keys in a group. | Yes |
| `Store.Delete` | `func (*Store) Delete(group, key string) error` | `dappco.re/go/core/io/store` | Delete removes a single key from a group. | Yes |
| `Store.DeleteGroup` | `func (*Store) DeleteGroup(group string) error` | `dappco.re/go/core/io/store` | DeleteGroup removes all keys in a group. | Yes |
| `Store.Get` | `func (*Store) Get(group, key string) (string, error)` | `dappco.re/go/core/io/store` | Get retrieves a value by group and key. | Yes |
| `Store.GetAll` | `func (*Store) GetAll(group string) (map[string]string, error)` | `dappco.re/go/core/io/store` | GetAll returns all key-value pairs in a group. | Yes |
| `Store.Render` | `func (*Store) Render(tmplStr, group string) (string, error)` | `dappco.re/go/core/io/store` | Render loads all key-value pairs from a group and renders a Go template. | Yes |
| `Store.Set` | `func (*Store) Set(group, key, value string) error` | `dappco.re/go/core/io/store` | Set stores a value by group and key, overwriting if exists. | Yes |
| `Service` | `type Service struct` | `dappco.re/go/core/io/workspace` | Service implements the Workspace interface. | Yes |
| `Workspace` | `type Workspace interface` | `dappco.re/go/core/io/workspace` | Workspace provides management for encrypted user workspaces. | No |
| `New` | `func New(c *core.Core, crypt ...cryptProvider) (any, error)` | `dappco.re/go/core/io/workspace` | New creates a new Workspace service instance. | Yes |
| `Service.CreateWorkspace` | `func (*Service) CreateWorkspace(identifier, password string) (string, error)` | `dappco.re/go/core/io/workspace` | CreateWorkspace creates a new encrypted workspace. | Yes |
| `Service.HandleIPCEvents` | `func (*Service) HandleIPCEvents(c *core.Core, msg core.Message) core.Result` | `dappco.re/go/core/io/workspace` | HandleIPCEvents handles workspace-related IPC messages. | No |
| `Service.SwitchWorkspace` | `func (*Service) SwitchWorkspace(name string) error` | `dappco.re/go/core/io/workspace` | SwitchWorkspace changes the active workspace. | Yes |
| `Service.WorkspaceFileGet` | `func (*Service) WorkspaceFileGet(filename string) (string, error)` | `dappco.re/go/core/io/workspace` | WorkspaceFileGet retrieves the content of a file from the active workspace. | Yes |
| `Service.WorkspaceFileSet` | `func (*Service) WorkspaceFileSet(filename, content string) error` | `dappco.re/go/core/io/workspace` | WorkspaceFileSet saves content to a file in the active workspace. | Yes |
| `Workspace.CreateWorkspace` | `CreateWorkspace(identifier, password string) (string, error)` | `dappco.re/go/core/io/workspace` | Creates a new encrypted workspace and returns its ID. | Yes |
| `Workspace.SwitchWorkspace` | `SwitchWorkspace(name string) error` | `dappco.re/go/core/io/workspace` | Switches the active workspace. | Yes |
| `Workspace.WorkspaceFileGet` | `WorkspaceFileGet(filename string) (string, error)` | `dappco.re/go/core/io/workspace` | Reads a file from the active workspace. | Yes |
| `Workspace.WorkspaceFileSet` | `WorkspaceFileSet(filename, content string) error` | `dappco.re/go/core/io/workspace` | Writes a file into the active workspace. | Yes |
