package main

import "testing"

func Test_Calc(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		arg  []int64
		want Stats
	}{
		{
			name: "empty slice",
			arg:  []int64{},
			want: Stats{},
		},
		{
			name: "single element",
			arg:  []int64{42},
			want: Stats{Count: 1, Sum: 42, Min: 42, Max: 42},
		},
		{
			name: "multiple elements",
			arg:  []int64{1, 2, 3, 4, 5},
			want: Stats{Count: 5, Sum: 15, Min: 1, Max: 5},
		},
		{
			name: "negative and positive elements",
			arg:  []int64{-10, 0, 10, 20},
			want: Stats{Count: 4, Sum: 20, Min: -10, Max: 20},
		},
		{
			name: "all negative elements",
			arg:  []int64{-5, -1, -3, -4, -2},
			want: Stats{Count: 5, Sum: -15, Min: -5, Max: -1},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := Calc(tt.arg)
			if got != tt.want {
				t.Fatalf("Calc(%v) = %+v; want %+v", tt.arg, got, tt.want)
			}
		})
	}
}
