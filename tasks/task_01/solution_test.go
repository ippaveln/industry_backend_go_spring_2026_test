package main

import (
	"testing"
)

func Test_greet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		arg  string
		want string
	}{
		{
			name: "empty name",
			arg:  "",
			want: "Hello, World!",
		},
		{
			name: "non-empty name",
			arg:  "Alice",
			want: "Hello, Alice!",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := greet(tt.arg)
			if got != tt.want {
				t.Fatalf("greet(%q) = %q; want %q", tt.arg, got, tt.want)
			}
		})
	}
}
