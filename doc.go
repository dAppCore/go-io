// Package io defines the storage boundary used across CoreGO.
//
//	medium, _ := io.NewSandboxed("/srv/app")
//	_ = medium.Write("config/app.yaml", "port: 8080")
//	backup, _ := io.NewSandboxed("/srv/backup")
//	_ = io.Copy(medium, "data/report.json", backup, "daily/report.json")
//
// Callers work against Medium so the same code can read and write state from
// sandboxed local paths, in-memory nodes, SQLite, S3, or other backends
// without changing application logic.
package io
