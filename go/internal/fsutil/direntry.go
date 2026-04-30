package fsutil

import "io/fs"

// SortDirEntriesByName sorts directory entries by their Name value.
func SortDirEntriesByName(entries []fs.DirEntry) {
	for i := 1; i < len(entries); i++ {
		entry := entries[i]
		j := i - 1
		for j >= 0 && entries[j].Name() > entry.Name() {
			entries[j+1] = entries[j]
			j--
		}
		entries[j+1] = entry
	}
}
