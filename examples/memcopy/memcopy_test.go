package main

import "testing"

type StructCopy2048 struct {
	Arr [2048]int8
}

//go:noinline
func DoCopy2048(s1, s2 *StructCopy2048) {
	*s1 = *s2
}

func BenchmarkCopy2048(b *testing.B) {
	s1 := StructCopy2048{}
	s2 := StructCopy2048{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DoCopy2048(&s1, &s2)
	}
}

type StructCopy4096 struct {
	Arr [4096]int8
}

//go:noinline
func DoCopy4096(s1, s2 *StructCopy4096) {
	*s1 = *s2
}

func BenchmarkCopy4096(b *testing.B) {
	s1 := StructCopy4096{}
	s2 := StructCopy4096{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DoCopy4096(&s1, &s2)
	}
}
