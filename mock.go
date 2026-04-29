// SPDX-License-Identifier: EUPL-1.2

package io

import (
	"cmp"
	core "dappco.re/go"
	goio "io"
	"io/fs"
	"slices"
	"sync" // Note: AX-6 — internal concurrency primitive; structural per RFC §5.1
	"time"
)

// MockMedium is an in-memory Medium implementation for testing.
// Tracks files, directories, and modification times without touching disk.
//
// Example:
//
//	mock := io.NewMockMedium()
//	_ = mock.Write("config/app.yaml", "port: 8080")
//	content, _ := mock.Read("config/app.yaml")
type MockMedium struct {
	mu sync.RWMutex
	// Files is the file content store. Exported for test assertions and direct writes.
	//
	//  mock.Files["config.yaml"] = "port: 8080"
	//  content := mock.Files["config.yaml"]
	Files map[string]string
	meta  map[string]mockMeta
	dirs  map[string]bool
}

type mockMeta struct {
	mode    fs.FileMode
	modTime time.Time
}

// NewMockMedium creates an empty in-memory Medium.
//
// Example:
//
//	mock := io.NewMockMedium()
func NewMockMedium() *MockMedium {
	return &MockMedium{
		Files: make(map[string]string),
		meta:  make(map[string]mockMeta),
		dirs:  make(map[string]bool),
	}
}

var _ Medium = (*MockMedium)(nil)

func (m *MockMedium) Read(path string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	content, ok := m.Files[path]
	if !ok {
		return "", fs.ErrNotExist
	}
	return content, nil
}

func (m *MockMedium) Write(path, content string) error {
	return m.WriteMode(path, content, 0644)
}

func (m *MockMedium) WriteMode(path, content string, mode fs.FileMode) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Files[path] = content
	m.meta[path] = mockMeta{mode: mode, modTime: time.Now()}
	return nil
}

func (m *MockMedium) EnsureDir(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.dirs[path] = true
	return nil
}

func (m *MockMedium) IsFile(path string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.Files[path]
	return ok
}

func (m *MockMedium) Delete(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.Files[path]; ok {
		delete(m.Files, path)
		delete(m.meta, path)
		return nil
	}
	if _, ok := m.dirs[path]; ok {
		delete(m.dirs, path)
		return nil
	}
	return fs.ErrNotExist
}

func (m *MockMedium) DeleteAll(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	found := false
	for k := range m.Files {
		if pathMatchesPrefix(k, path) {
			delete(m.Files, k)
			delete(m.meta, k)
			found = true
		}
	}
	for d := range m.dirs {
		if pathMatchesPrefix(d, path) {
			delete(m.dirs, d)
			found = true
		}
	}
	if !found {
		return fs.ErrNotExist
	}
	return nil
}

func (m *MockMedium) Rename(oldPath, newPath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	f, ok := m.Files[oldPath]
	if !ok {
		return fs.ErrNotExist
	}
	m.Files[newPath] = f
	delete(m.Files, oldPath)
	if metadata, ok := m.meta[oldPath]; ok {
		m.meta[newPath] = metadata
		delete(m.meta, oldPath)
	}
	return nil
}

func (m *MockMedium) List(path string) ([]fs.DirEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	prefix := mockListPrefix(path)
	seen := make(map[string]bool)
	entries := make([]fs.DirEntry, 0)
	entries = append(entries, m.fileEntries(prefix, seen)...)
	entries = append(entries, m.dirEntries(prefix, seen)...)
	slices.SortFunc(entries, func(a, b fs.DirEntry) int { return cmp.Compare(a.Name(), b.Name()) })
	return entries, nil
}

func pathMatchesPrefix(candidate, prefix string) bool {
	return candidate == prefix || len(candidate) > len(prefix) && candidate[:len(prefix)+1] == prefix+"/"
}

func mockListPrefix(filePath string) string {
	if filePath == "" || filePath == "." {
		return ""
	}
	return filePath + "/"
}

func (m *MockMedium) fileEntries(prefix string, seen map[string]bool) []fs.DirEntry {
	var entries []fs.DirEntry
	for k, content := range m.Files {
		if len(k) <= len(prefix) || k[:len(prefix)] != prefix {
			continue
		}
		rest := k[len(prefix):]
		if dirName, ok := firstPathComponent(rest); ok {
			entries = appendMockDirectoryEntry(entries, seen, dirName)
			continue
		}
		if !seen[rest] {
			seen[rest] = true
			mt := m.meta[k]
			entries = append(entries, NewDirEntry(rest, false, mt.mode, NewFileInfo(rest, int64(len(content)), mt.mode, mt.modTime, false)))
		}
	}
	return entries
}

func (m *MockMedium) dirEntries(prefix string, seen map[string]bool) []fs.DirEntry {
	var entries []fs.DirEntry
	for d := range m.dirs {
		if len(d) <= len(prefix) || d[:len(prefix)] != prefix {
			continue
		}
		rest := d[len(prefix):]
		if dirName, ok := firstPathComponent(rest); ok {
			entries = appendMockDirectoryEntry(entries, seen, dirName)
			continue
		}
		entries = appendMockDirectoryEntry(entries, seen, rest)
	}
	return entries
}

func firstPathComponent(rest string) (string, bool) {
	for i, c := range rest {
		if c == '/' {
			return rest[:i], true
		}
	}
	return "", false
}

func appendMockDirectoryEntry(entries []fs.DirEntry, seen map[string]bool, name string) []fs.DirEntry {
	if name == "" || seen[name] {
		return entries
	}
	seen[name] = true
	return append(entries, NewDirEntry(name, true, 0755, NewFileInfo(name, 0, 0755, time.Now(), true)))
}

func (m *MockMedium) Stat(path string) (fs.FileInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if content, ok := m.Files[path]; ok {
		mt := m.meta[path]
		return NewFileInfo(core.PathBase(path), int64(len(content)), mt.mode, mt.modTime, false), nil
	}
	if m.dirs[path] {
		return NewFileInfo(core.PathBase(path), 0, 0755, time.Now(), true), nil
	}
	return nil, fs.ErrNotExist
}

func (m *MockMedium) Open(path string) (fs.File, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	content, ok := m.Files[path]
	if !ok {
		return nil, fs.ErrNotExist
	}
	mt := m.meta[path]
	return &MockFile{reader: core.NewReader(content), info: NewFileInfo(core.PathBase(path), int64(len(content)), mt.mode, mt.modTime, false)}, nil
}

func (m *MockMedium) Create(path string) (goio.WriteCloser, error) {
	return &MockWriteCloser{medium: m, path: path}, nil
}

func (m *MockMedium) Append(path string) (goio.WriteCloser, error) {
	m.mu.RLock()
	existing := m.Files[path]
	m.mu.RUnlock()
	return &MockWriteCloser{medium: m, path: path, data: []byte(existing)}, nil
}

func (m *MockMedium) ReadStream(path string) (goio.ReadCloser, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	f, ok := m.Files[path]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return goio.NopCloser(core.NewReader(f)), nil
}

func (m *MockMedium) WriteStream(path string) (goio.WriteCloser, error) {
	return m.Create(path)
}

func (m *MockMedium) Exists(path string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, fileOK := m.Files[path]
	return fileOK || m.dirs[path]
}

func (m *MockMedium) IsDir(path string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.dirs[path]
}

// MockFile implements fs.File for MockMedium.
//
// Example:
//
//	file, _ := mock.Open("config/app.yaml")
//	defer file.Close()
type MockFile struct {
	reader goio.Reader
	info   fs.FileInfo
}

func (f *MockFile) Read(p []byte) (int, error) { return f.reader.Read(p) }
func (f *MockFile) Stat() (fs.FileInfo, error) { return f.info, nil }
func (f *MockFile) Close() error               { return nil }

// MockWriteCloser implements io.WriteCloser for MockMedium.
// On Close, the buffered content is written to the mock filesystem.
//
// Example:
//
//	w, _ := mock.Create("output.txt")
//	w.Write([]byte("hello"))
//	w.Close()
type MockWriteCloser struct {
	medium *MockMedium
	path   string
	data   []byte
}

func (w *MockWriteCloser) Write(p []byte) (int, error) {
	w.data = append(w.data, p...)
	return len(p), nil
}

func (w *MockWriteCloser) Close() error {
	return w.medium.Write(w.path, string(w.data))
}
