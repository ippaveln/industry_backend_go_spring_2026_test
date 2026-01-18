package main

import (
	"testing"
)

func Test_fizzBuzz(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		arg     int
		want    string
		wantErr bool
	}{
		{
			name:    "divisible by 3",
			arg:     9,
			want:    "Fizz",
			wantErr: false,
		},
		{
			name:    "divisible by 5",
			arg:     10,
			want:    "Buzz",
			wantErr: false,
		},
		{
			name:    "divisible by 3 and 5",
			arg:     15,
			want:    "FizzBuzz",
			wantErr: false,
		},
		{
			name:    "not divisible by 3 or 5",
			arg:     7,
			want:    "7",
			wantErr: false,
		},
		{
			name:    "negative number",
			arg:     -5,
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := fizzBuzz(tt.arg)
			if got != tt.want {
				t.Fatalf("fizzBuzz(%d) = %q; want %q", tt.arg, got, tt.want)
			}
			if (err != nil) != tt.wantErr {
				t.Fatalf("fizzBuzz(%d) error = %v; wantErr %v", tt.arg, err, tt.wantErr)
			}
		})
	}
}
