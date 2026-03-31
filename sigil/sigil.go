// Example: hexSigil, _ := sigil.NewSigil("hex")
// Example: gzipSigil, _ := sigil.NewSigil("gzip")
// Example: encoded, _ := sigil.Transmute([]byte("payload"), []sigil.Sigil{hexSigil, gzipSigil})
// Example: decoded, _ := sigil.Untransmute(encoded, []sigil.Sigil{hexSigil, gzipSigil})
package sigil

import core "dappco.re/go/core"

type Sigil interface {
	// Example: encoded, _ := hexSigil.In([]byte("payload"))
	In(data []byte) ([]byte, error)

	// Example: decoded, _ := hexSigil.Out(encoded)
	Out(data []byte) ([]byte, error)
}

// Example: encoded, _ := sigil.Transmute([]byte("payload"), []sigil.Sigil{hexSigil, gzipSigil})
func Transmute(data []byte, sigils []Sigil) ([]byte, error) {
	var err error
	for _, sigilValue := range sigils {
		data, err = sigilValue.In(data)
		if err != nil {
			return nil, core.E("sigil.Transmute", "sigil in failed", err)
		}
	}
	return data, nil
}

// Example: decoded, _ := sigil.Untransmute(encoded, []sigil.Sigil{hexSigil, gzipSigil})
func Untransmute(data []byte, sigils []Sigil) ([]byte, error) {
	var err error
	for i := len(sigils) - 1; i >= 0; i-- {
		data, err = sigils[i].Out(data)
		if err != nil {
			return nil, core.E("sigil.Untransmute", "sigil out failed", err)
		}
	}
	return data, nil
}
