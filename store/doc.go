// Package store provides a SQLite-backed group-namespaced key-value store.
//
//	kvStore, _ := store.New(store.Options{Path: ":memory:"})
//	_ = kvStore.Set("app", "theme", "midnight")
//	medium := kvStore.AsMedium()
//	_ = medium.Write("app/theme", "midnight")
//
// It also exposes an io.Medium adapter so grouped values can participate in
// the same storage workflows as filesystem-backed mediums.
package store
