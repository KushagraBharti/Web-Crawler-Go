package crawler

import (
	"strings"
	"testing"
	"time"
)

func TestReadBodyLimited(t *testing.T) {
	data, size, errClass := readBodyLimited(strings.NewReader("hello"), 4)
	if errClass != ErrSizeLimit {
		t.Fatalf("expected size limit, got %s", errClass)
	}
	if size <= 4 {
		t.Fatalf("expected size > 4, got %d", size)
	}
	if data != nil {
		t.Fatal("expected nil data on size limit")
	}
}

func TestParseRetryAfter(t *testing.T) {
	if d := parseRetryAfter(""); d != 0 {
		t.Fatal("expected zero")
	}
	if d := parseRetryAfter("5"); d != 5*time.Second {
		t.Fatalf("expected 5s, got %v", d)
	}
}