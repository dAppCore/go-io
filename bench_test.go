package io

import (
	"testing"
)

const benchTestPath = "test.txt"

func BenchmarkMemoryMedium_Write(b *testing.B) {
	medium := NewMemoryMedium()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = medium.Write(benchTestPath, "some content")
	}
}

func BenchmarkMemoryMedium_Read(b *testing.B) {
	medium := NewMemoryMedium()
	_ = medium.Write(benchTestPath, "some content")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = medium.Read(benchTestPath)
	}
}

func BenchmarkMemoryMedium_List(b *testing.B) {
	medium := NewMemoryMedium()
	_ = medium.EnsureDir("dir")
	for i := 0; i < 100; i++ {
		_ = medium.Write("dir/file"+string(rune(i))+".txt", "content")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = medium.List("dir")
	}
}
