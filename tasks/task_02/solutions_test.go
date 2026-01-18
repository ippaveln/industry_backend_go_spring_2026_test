package main

import "testing"

func Test_reverseRunes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		arg  string
		want string
	}{
		{
			name: "empty string",
			arg:  "",
			want: "",
		},
		{
			name: "single rune",
			arg:  "A",
			want: "A",
		},
		{
			name: "ASCII string",
			arg:  "Hello, World!",
			want: "!dlroW ,olleH",
		},
		{
			name: "UTF-8 string",
			arg:  "АБОБА",
			want: "АБОБА",
		},
		{
			name: "UTF-8 string - 2",
			arg:  "АБООБА",
			want: "АБООБА",
		},
		{
			name: "UTF-8 string - 3",
			arg:  "АБВГДЕ",
			want: "ЕДГВБА",
		},
		{
			name: "mixed string",
			arg:  "Hello, 世界",
			want: "界世 ,olleH",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := reverseRunes(tt.arg)
			if got != tt.want {
				t.Fatalf("reverseRunes(%q) = %q; want %q", tt.arg, got, tt.want)
			}
		})
	}
}
