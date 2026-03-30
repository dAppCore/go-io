// Package io defines the storage abstraction used across CoreGO.
//
// Callers work against Medium so the same code can read and write state from
// sandboxed local paths, in-memory nodes, SQLite, S3, or other backends
// without changing application logic.
package io
