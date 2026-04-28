package pwa

import (
	"errors"
	goio "io"
	"io/fs"

	core "dappco.re/go"
	coreio "dappco.re/go/io"
)

// PWA Medium is intentionally stubbed pending two Snider crypto-trio deps:
//   - forge.lthn.ai/Snider/Borg — Borg IS the PWA collector (headless-browser
//     scraping is one of Borg's many roles). It also wraps the scraped
//     artefact in a DataNode. io.Medium was designed FOR DataNodes from day 1.
//   - forge.lthn.ai/Snider/Enchantrix — encrypts the fetched payload at rest.
//     Trixxie is THE encryption layer for ALL encryption across the stack.
// Borg's PWA collector is the active surface here — pwa.go is just the
//   Medium-interface wrapper. Wire BOTH at canonical forge.lthn.ai/Snider/*
//   paths when scaffolded — never migrate to dappco.re/*.

// ErrNotImplemented is returned by all error-returning operations while PWA
// Medium is stubbed.
var ErrNotImplemented = errors.New("pwa medium is not implemented")

// Medium is a stub PWA-backed implementation of coreio.Medium.
type Medium struct {
	url string
}

var _ coreio.Medium = (*Medium)(nil)

// Options configures a PWA Medium.
type Options struct {
	URL string
}

// New creates a stub PWA Medium.
func New(options Options) (*Medium, error) {
	return &Medium{url: options.URL}, nil
}

func notImplemented(operation string) error {
	return core.E(operation, "PWA medium is not implemented", ErrNotImplemented)
}

// Read returns ErrNotImplemented while the PWA collector wiring is stubbed.
func (medium *Medium) Read(filePath string) (string, error) {
	return "", notImplemented("pwa.Read")
}

// Write returns ErrNotImplemented while the PWA collector wiring is stubbed.
func (medium *Medium) Write(filePath, content string) error {
	return notImplemented("pwa.Write")
}

// WriteMode returns ErrNotImplemented while the PWA collector wiring is stubbed.
func (medium *Medium) WriteMode(filePath, content string, mode fs.FileMode) error {
	return notImplemented("pwa.WriteMode")
}

// EnsureDir returns ErrNotImplemented while the PWA collector wiring is stubbed.
func (medium *Medium) EnsureDir(filePath string) error {
	return notImplemented("pwa.EnsureDir")
}

// IsFile reports false while the PWA collector wiring is stubbed.
func (medium *Medium) IsFile(filePath string) bool {
	return false
}

// Delete returns ErrNotImplemented while the PWA collector wiring is stubbed.
func (medium *Medium) Delete(filePath string) error {
	return notImplemented("pwa.Delete")
}

// DeleteAll returns ErrNotImplemented while the PWA collector wiring is stubbed.
func (medium *Medium) DeleteAll(filePath string) error {
	return notImplemented("pwa.DeleteAll")
}

// Rename returns ErrNotImplemented while the PWA collector wiring is stubbed.
func (medium *Medium) Rename(oldPath, newPath string) error {
	return notImplemented("pwa.Rename")
}

// List returns ErrNotImplemented while the PWA collector wiring is stubbed.
func (medium *Medium) List(filePath string) ([]fs.DirEntry, error) {
	return nil, notImplemented("pwa.List")
}

// Stat returns ErrNotImplemented while the PWA collector wiring is stubbed.
func (medium *Medium) Stat(filePath string) (fs.FileInfo, error) {
	return nil, notImplemented("pwa.Stat")
}

// Open returns ErrNotImplemented while the PWA collector wiring is stubbed.
func (medium *Medium) Open(filePath string) (fs.File, error) {
	return nil, notImplemented("pwa.Open")
}

// Create returns ErrNotImplemented while the PWA collector wiring is stubbed.
func (medium *Medium) Create(filePath string) (goio.WriteCloser, error) {
	return nil, notImplemented("pwa.Create")
}

// Append returns ErrNotImplemented while the PWA collector wiring is stubbed.
func (medium *Medium) Append(filePath string) (goio.WriteCloser, error) {
	return nil, notImplemented("pwa.Append")
}

// ReadStream returns ErrNotImplemented while the PWA collector wiring is stubbed.
func (medium *Medium) ReadStream(filePath string) (goio.ReadCloser, error) {
	return nil, notImplemented("pwa.ReadStream")
}

// WriteStream returns ErrNotImplemented while the PWA collector wiring is stubbed.
func (medium *Medium) WriteStream(filePath string) (goio.WriteCloser, error) {
	return nil, notImplemented("pwa.WriteStream")
}

// Exists reports false while the PWA collector wiring is stubbed.
func (medium *Medium) Exists(filePath string) bool {
	return false
}

// IsDir reports false while the PWA collector wiring is stubbed.
func (medium *Medium) IsDir(filePath string) bool {
	return false
}
