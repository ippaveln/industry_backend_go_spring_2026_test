package main

import (
	"testing"
)

func TestHello_ReturnsExactGreeting(t *testing.T) {
	t.Parallel()

	got := hello()
	want := "Hello, World!"
	if got != want {
		t.Fatalf("hello() = %q; want %q", got, want)
	}
}
