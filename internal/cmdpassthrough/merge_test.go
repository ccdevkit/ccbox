package cmdpassthrough

import (
	"testing"
)

func TestMerge_AppendsRatherThanReplaces(t *testing.T) {
	a := []string{"cmd1", "cmd2"}
	b := []string{"cmd3", "cmd4"}
	got := Merge(a, b)
	want := []string{"cmd1", "cmd2", "cmd3", "cmd4"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestMerge_EmptySources(t *testing.T) {
	got := Merge()
	if got == nil {
		t.Fatal("expected non-nil empty slice, got nil")
	}
	if len(got) != 0 {
		t.Fatalf("expected empty slice, got %v", got)
	}
}

func TestMerge_MultipleSources(t *testing.T) {
	a := []string{"a"}
	b := []string{"b", "c"}
	c := []string{"d"}
	got := Merge(a, b, c)
	want := []string{"a", "b", "c", "d"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestMerge_NilSlicesHandledGracefully(t *testing.T) {
	got := Merge(nil, []string{"a"}, nil, []string{"b"})
	want := []string{"a", "b"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestMerge_Deduplication(t *testing.T) {
	a := []string{"cmd1", "cmd2"}
	b := []string{"cmd2", "cmd3"}
	got := Merge(a, b)
	want := []string{"cmd1", "cmd2", "cmd3"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
