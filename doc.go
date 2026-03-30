// Package io gives CoreGO a single storage surface.
//
//	medium, _ := io.NewSandboxed("/srv/app")
//	_ = medium.Write("config/app.yaml", "port: 8080")
//	backup, _ := io.NewSandboxed("/srv/backup")
//	_ = io.Copy(medium, "data/report.json", backup, "daily/report.json")
package io
