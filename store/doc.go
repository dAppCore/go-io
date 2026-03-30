// Package store provides a SQLite-backed group-namespaced key-value store.
//
//	keyValueStore, _ := store.New(store.Options{Path: ":memory:"})
//	_ = keyValueStore.Set("app", "theme", "midnight")
//	medium := keyValueStore.AsMedium()
//	_ = medium.Write("app/theme", "midnight")
//
// It also exposes an io.Medium adapter so grouped values can participate in
// the same storage workflows as filesystem-backed mediums.
package store
