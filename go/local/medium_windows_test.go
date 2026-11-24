// SPDX-License-Identifier: EUPL-1.2

//go:build windows

package local

import core "github.com/dappcore/go"

// A medium over the whole filesystem ("/", which is C:\ on Windows) takes
// absolute paths on its own drive: what io.Local and config.Save do with
// %AppData%\App\settings.yaml.
func TestLocal_WindowsDrive_Good(t *core.T) {
	m, err := New("/")
	core.RequireNoError(t, err)
	defer m.Close()
	dir := core.PathJoin(t.TempDir(), "app")

	core.RequireNoError(t, m.EnsureDir(dir))
	core.RequireNoError(t, m.Write(core.PathJoin(dir, "settings.yaml"), "keep_days: 30\n"))

	got, err := m.Read(core.PathJoin(dir, "settings.yaml"))
	core.RequireNoError(t, err)
	core.AssertEqual(t, "keep_days: 30\n", got)
}

// A path on another drive can't be reached through a medium rooted on this
// one: refused, not silently written somewhere on C:.
func TestLocal_WindowsDrive_Bad(t *core.T) {
	m, err := New("/")
	core.RequireNoError(t, err)
	defer m.Close()
	other := "Q:"
	if core.EqualFold(driveOf(core.PathToSlash(m.filesystemRoot)), "Q:") {
		other = "R:"
	}

	core.AssertError(t, m.Write(other+`\elsewhere\file.txt`, "x"))
}

// The drive letter matches whatever its case, as Windows does.
func TestLocal_WindowsDrive_Ugly(t *core.T) {
	m, err := New("/")
	core.RequireNoError(t, err)
	defer m.Close()
	path := core.PathJoin(t.TempDir(), "lower.txt")
	lower := core.Lower(path[:1]) + path[1:]

	core.RequireNoError(t, m.Write(lower, "ok"))
	core.AssertTrue(t, m.IsFile(path))
}
