// Example: medium, _ := io.NewSandboxed("/srv/app")
// Example: _ = medium.Write("config/app.yaml", "port: 8080")
// Example: backup, _ := io.NewSandboxed("/srv/backup")
// Example: _ = io.Copy(medium, "data/report.json", backup, "daily/report.json")
package io
