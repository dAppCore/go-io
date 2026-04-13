// SPDX-License-Identifier: EUPL-1.2

package io

import (
	"bytes"
	"cmp"
	goio "io"
	"io/fs"
	"slices"
	"sync"
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
	delete(m.Files, path)
	return nil
}

func (m *MockMedium) DeleteAll(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k := range m.Files {
		if k == path || len(k) > len(path) && k[:len(path)+1] == path+"/" {
			delete(m.Files, k)
		}
	}
	delete(m.dirs, path)
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
	return nil
}

func (m *MockMedium) List(path string) ([]fs.DirEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	prefix := path + "/"
	if path == "" || path == "." {
		prefix = ""
	}
	seen := make(map[string]bool)
	var entries []fs.DirEntry
	for k, content := range m.Files {
		if len(k) <= len(prefix) || k[:len(prefix)] != prefix {
			continue
		}
		rest := k[len(prefix):]
		slash := -1
		for i, c := range rest {
			if c == '/' {
				slash = i
				break
			}
		}
		if slash >= 0 {
			dirName := rest[:slash]
			if !seen[dirName] {
				seen[dirName] = true
				entries = append(entries, NewDirEntry(dirName, true, 0755, NewFileInfo(dirName, 0, 0755, time.Now(), true)))
			}
		} else {
			if !seen[rest] {
				seen[rest] = true
				mt := m.meta[k]
				entries = append(entries, NewDirEntry(rest, false, mt.mode, NewFileInfo(rest, int64(len(content)), mt.mode, mt.modTime, false)))
			}
		}
	}
	slices.SortFunc(entries, func(a, b fs.DirEntry) int { return cmp.Compare(a.Name(), b.Name()) })
	return entries, nil
}

func (m *MockMedium) Stat(path string) (fs.FileInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if content, ok := m.Files[path]; ok {
		mt := m.meta[path]
		return NewFileInfo(path, int64(len(content)), mt.mode, mt.modTime, false), nil
	}
	if m.dirs[path] {
		return NewFileInfo(path, 0, 0755, time.Now(), true), nil
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
	return &MockFile{Reader: bytes.NewReader([]byte(content)), info: NewFileInfo(path, int64(len(content)), mt.mode, mt.modTime, false)}, nil
}

func (m *MockMedium) Create(path string) (goio.WriteCloser, error) {
	return &MockWriteCloser{medium: m, path: path}, nil
}

func (m *MockMedium) Append(path string) (goio.WriteCloser, error) {
	m.mu.RLock()
	existing := m.Files[path]
	m.mu.RUnlock()
	return &MockWriteCloser{medium: m, path: path, buf: *bytes.NewBufferString(existing)}, nil
}

func (m *MockMedium) ReadStream(path string) (goio.ReadCloser, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	f, ok := m.Files[path]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return goio.NopCloser(bytes.NewReader([]byte(f))), nil
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
	*bytes.Reader
	info fs.FileInfo
}

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
	buf    bytes.Buffer
}

func (w *MockWriteCloser) Write(p []byte) (int, error) { return w.buf.Write(p) }

func (w *MockWriteCloser) Close() error {
	return w.medium.Write(w.path, w.buf.String())
}
