package trial

import (
	"testing"
)

func TestAdd(t *testing.T) {
	tests := []struct {
		a, b int
		want int
	}{
		{1, 2, 3},
		{3, 4, 7},
		{4, 5, 9},
	}
	for _, tc := range tests {
		got := Add(tc.a, tc.b)
		if got != tc.want {
			t.Errorf("add(%d,%d)=%d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}
