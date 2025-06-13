package main

import "testing"

type StructZero2048 struct {
	Arr [2048]int8
}

//go:noinline
func DoZero2048(s *StructZero2048) {
	*s = StructZero2048{}
}

func BenchmarkZero2048(b *testing.B) {
	s := StructZero2048{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DoZero2048(&s)
	}
}

type StructZero4096 struct {
	Arr [4096]int8
}

//go:noinline
func DoZero4096(s *StructZero4096) {
	*s = StructZero4096{}
}

func BenchmarkZero4096(b *testing.B) {
	s := StructZero4096{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DoZero4096(&s)
	}
}
