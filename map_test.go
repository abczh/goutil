package goutil

import (
	"testing"
)

func TestRwMap(t *testing.T) {
	m := RwMap(1000)
	m.Store(1, "a")
	m.Store(2, "b")
	m.Store(3, "c")
	m.Store(4, "d")
	m.Store(5, "e")
	m.Store(6, "f")
	t.Logf("Len: %d", m.Len())
	var s = make(map[interface{}]int)
	for i := 10000; i > 0; i-- {
		k, _, _ := m.Random()
		s[k]++
	}
	t.Logf("%#v", s)
}

func TestAtomicMap(t *testing.T) {
	m := AtomicMap()
	m.Store(1, "a")
	m.Store(2, "b")
	m.Store(3, "c")
	m.Store(4, "d")
	m.Store(5, "e")
	m.Store(6, "f")
	t.Logf("Len: %d", m.Len())
	var s = make(map[interface{}]int)
	for i := 10000; i > 0; i-- {
		k, _, _ := m.Random()
		s[k]++
	}
	t.Logf("%#v", s)
}
