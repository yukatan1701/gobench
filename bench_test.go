package main

import "testing"

type StructCopy struct {
	Arr [4096]int8
}

//go:noinline
func DoCopy(s1, s2 *StructCopy) {
	*s1 = *s2
}

func BenchmarkCopy(b *testing.B) {
	s1 := StructCopy{}
	s2 := StructCopy{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DoCopy(&s1, &s2)
	}
}
