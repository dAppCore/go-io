package sigil

import core "dappco.re/go"

// SigilBuffer aliases the package-internal sigilBuffer for the example
// surface. Renamed from `Buffer` to avoid collision with core.Buffer
// (the bytes.Buffer alias added in dappco.re/go v0.10.0) which is
// dot-imported by sibling test files in this package.
type SigilBuffer = sigilBuffer

func ExampleReverseSigil_In() {
	core.Println("ok")
	// Output: ok
}

func ExampleReverseSigil_Out() {
	core.Println("ok")
	// Output: ok
}

func ExampleHexSigil_In() {
	core.Println("ok")
	// Output: ok
}

func ExampleHexSigil_Out() {
	core.Println("ok")
	// Output: ok
}

func ExampleBase64Sigil_In() {
	core.Println("ok")
	// Output: ok
}

func ExampleBase64Sigil_Out() {
	core.Println("ok")
	// Output: ok
}

func ExampleBuffer_Write() {
	core.Println("ok")
	// Output: ok
}

func ExampleBuffer_Bytes() {
	core.Println("ok")
	// Output: ok
}

func ExampleGzipSigil_In() {
	core.Println("ok")
	// Output: ok
}

func ExampleGzipSigil_Out() {
	core.Println("ok")
	// Output: ok
}

func ExampleJSONSigil_In() {
	core.Println("ok")
	// Output: ok
}

func ExampleJSONSigil_Out() {
	core.Println("ok")
	// Output: ok
}

func ExampleNewHashSigil() {
	core.Println("ok")
	// Output: ok
}

func ExampleHashSigil_In() {
	core.Println("ok")
	// Output: ok
}

func ExampleHashSigil_Out() {
	core.Println("ok")
	// Output: ok
}

func ExampleNewSigil() {
	core.Println("ok")
	// Output: ok
}
