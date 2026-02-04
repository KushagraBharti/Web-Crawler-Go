package crawler

import "testing"

func TestDeduperSeen(t *testing.T) {
	d := NewDeduper(4)
	if d.Seen("a") {
		t.Fatal("first seen should be false")
	}
	if !d.Seen("a") {
		t.Fatal("second seen should be true")
	}
	if d.Seen("") {
		t.Fatal("empty key should be treated as seen")
	}
}