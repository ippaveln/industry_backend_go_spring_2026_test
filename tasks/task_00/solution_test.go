package main

import (
	"testing"
)

func Test_greet(t *testing.T) {
	t.Parallel()

	got := greet()
	want := "Hello, World!"
	if got != want {
		t.Fatalf("hello() = %q; want %q", got, want)
	}
}
