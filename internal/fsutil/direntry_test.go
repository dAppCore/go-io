package fsutil

import (
	core "dappco.re/go"
	"io/fs"
)

type sortTestDirEntry struct{ name string }

func (entry sortTestDirEntry) Name() string               { return entry.name }
func (entry sortTestDirEntry) IsDir() bool                { return false }
func (entry sortTestDirEntry) Type() fs.FileMode          { return 0 }
func (entry sortTestDirEntry) Info() (fs.FileInfo, error) { return nil, nil }

func TestDirentry_SortDirEntriesByName_Good(t *core.T) {
	entries := []fs.DirEntry{sortTestDirEntry{name: "b"}, sortTestDirEntry{name: "a"}}
	SortDirEntriesByName(entries)
	core.AssertEqual(t, "a", entries[0].Name())
}

func TestDirentry_SortDirEntriesByName_Bad(t *core.T) {
	entries := []fs.DirEntry(nil)
	SortDirEntriesByName(entries)
	core.AssertNil(t, entries)
}

func TestDirentry_SortDirEntriesByName_Ugly(t *core.T) {
	entries := []fs.DirEntry{sortTestDirEntry{name: ""}, sortTestDirEntry{name: "a"}}
	SortDirEntriesByName(entries)
	core.AssertEqual(t, "", entries[0].Name())
}
