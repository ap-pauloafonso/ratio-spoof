package printer

import (
	"fmt"
	"testing"
)

func TestHumanReadableSize(T *testing.T) {
	data := []struct {
		in  float64
		out string
	}{
		{1536, "1.50KiB"},
		{379040563, "361.48MiB"},
		{6291456, "6.00MiB"},
		{372749107, "355.48MiB"},
		{10485760, "10.00MiB"},
		{15728640, "15.00MiB"},
		{363311923, "346.48MiB"},
		{16777216, "16.00MiB"},
		{379040563, "361.48MiB"},
	}
	for idx, td := range data {
		T.Run(fmt.Sprint(idx), func(t *testing.T) {
			got := humanReadableSize(td.in)
			if got != td.out {
				t.Errorf("got %q, want %q", got, td.out)
			}
		})
	}
}
