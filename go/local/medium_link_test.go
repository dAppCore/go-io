package local

import (
	"syscall"

	core "dappco.re/go"
)

func TestMediumLink_lstat_Good(t *core.T) {
	path := core.Path(t.TempDir(), "regular.txt")
	core.RequireTrue(t, core.WriteFile(path, []byte("ready"), 0644).OK)

	info, err := lstat(path)

	core.AssertNoError(t, err)
	core.AssertFalse(t, info.IsSymlink)
	core.AssertNotEqual(t, core.FileMode(0), info.Mode)
	core.AssertEqual(t, int64(5), info.Size)
	core.AssertFalse(t, info.ModTime.IsZero())
}

func TestMediumLink_lstat_Bad(t *core.T) {
	_, err := lstat(core.Path(t.TempDir(), "missing.txt"))

	core.AssertError(t, err)
	core.AssertTrue(t, core.Is(err, syscall.ENOENT) || core.IsNotExist(err))
}

func TestMediumLink_lstat_Ugly(t *core.T) {
	dir := t.TempDir()
	target := core.Path(dir, "target.txt")
	link := core.Path(dir, "current.txt")
	core.RequireTrue(t, core.WriteFile(target, []byte("ready"), 0644).OK)
	if err := syscall.Symlink(target, link); err != nil {
		t.Skip(err)
	}

	info, err := lstat(link)

	core.AssertNoError(t, err)
	core.AssertTrue(t, info.IsSymlink)
	core.AssertTrue(t, info.Mode&core.ModeSymlink != 0)
}

func TestMediumLink_readlink_Good(t *core.T) {
	dir := t.TempDir()
	target := core.Path(dir, "target.txt")
	link := core.Path(dir, "current.txt")
	core.RequireTrue(t, core.WriteFile(target, []byte("ready"), 0644).OK)
	if err := syscall.Symlink(target, link); err != nil {
		t.Skip(err)
	}

	got, err := readlink(link)

	core.AssertNoError(t, err)
	core.AssertEqual(t, target, got)
}

func TestMediumLink_readlink_Bad(t *core.T) {
	path := core.Path(t.TempDir(), "regular.txt")
	core.RequireTrue(t, core.WriteFile(path, []byte("ready"), 0644).OK)

	_, err := readlink(path)

	core.AssertError(t, err)
}

func TestMediumLink_readlink_Ugly(t *core.T) {
	dir := t.TempDir()
	link := core.Path(dir, "long-current.txt")
	parts := make([]string, 40)
	for i := range parts {
		parts[i] = core.Sprintf("segment%02d", i)
	}
	target := core.Join("/", parts...)
	if err := syscall.Symlink(target, link); err != nil {
		t.Skip(err)
	}

	got, err := readlink(link)

	core.AssertNoError(t, err)
	core.AssertEqual(t, target, got)
}
