package input

import (
	"fmt"
	"testing"
)

func TestStrSize2ByteSize(T *testing.T) {

	data := []struct {
		in          string
		inTotalSize int
		out         int
	}{
		{"100kb", 100, 102400},
		{"1kb", 0, 1024},
		{"1mb", 0, 1048576},
		{"1gb", 0, 1073741824},
		{"1.5gb", 0, 1610612736},
		{"1tb", 0, 1099511627776},
		{"1b", 0, 1},
		{"100%", 10737418240, 10737418240},
		{"55%", 943718400, 519045120},
	}

	for idx, td := range data {
		T.Run(fmt.Sprint(idx), func(t *testing.T) {
			got := strSize2ByteSize(td.in, td.inTotalSize)
			if got != td.out {
				t.Errorf("got %v, want %v", got, td.out)
			}
		})
	}
}
