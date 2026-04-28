package fsutil

import (
	"io/fs"

	core "dappco.re/go"
)

type ax7DirEntry struct{ name string }

func (entry ax7DirEntry) Name() string               { return entry.name }
func (entry ax7DirEntry) IsDir() bool                { return false }
func (entry ax7DirEntry) Type() fs.FileMode          { return 0 }
func (entry ax7DirEntry) Info() (fs.FileInfo, error) { return nil, nil }

func TestAX7_SortDirEntriesByName_Good(t *core.T) {
	entries := []fs.DirEntry{ax7DirEntry{name: "b"}, ax7DirEntry{name: "a"}}
	SortDirEntriesByName(entries)
	core.AssertEqual(t, "a", entries[0].Name())
}

func TestAX7_SortDirEntriesByName_Bad(t *core.T) {
	entries := []fs.DirEntry(nil)
	SortDirEntriesByName(entries)
	core.AssertNil(t, entries)
}

func TestAX7_SortDirEntriesByName_Ugly(t *core.T) {
	entries := []fs.DirEntry{ax7DirEntry{name: ""}, ax7DirEntry{name: "a"}}
	SortDirEntriesByName(entries)
	core.AssertEqual(t, "", entries[0].Name())
}
