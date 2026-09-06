package provider

import "testing"

func TestNew(t *testing.T) {
	p := New("test")()
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
}
