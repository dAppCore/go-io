// Package store maps grouped keys onto SQLite rows.
//
//	keyValueStore, _ := store.New(store.Options{Path: ":memory:"})
//	_ = keyValueStore.Set("app", "theme", "midnight")
//	medium := keyValueStore.AsMedium()
//	_ = medium.Write("app/theme", "midnight")
package store
