package ratiospoof

import (
	"fmt"
	"testing"
)

func assertAreEqual(t *testing.T, got, want interface{}) {
	t.Helper()
	if got != want {
		t.Errorf("\ngot : %v\nwant: %v", got, want)
	}
}

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

func TestClculateNextTotalSizeByte(T *testing.T) {

	got := calculateNextTotalSizeByte(100*1024, 0, 512, 30, 87979879)
	want := 3075072

	assertAreEqual(T, got, want)
}

// func TestUrlEncodeInfoHash(T *testing.T) {

// 	b, _ := ioutil.ReadFile("")
// 	got := extractInfoHashURLEncoded(b, beencode.Decode(b))
// 	want := "%60N%7d%1f%8b%3a%9bT%d5%fc%ad%d1%27%ab5%02%1c%fb%03%b0"
// 	assertAreEqual(T, got, want)
// }

// func TestUrlEncodeInfoHash2(T *testing.T) {

// 	b, _ := ioutil.ReadFile("")
// 	got := extractInfoHashURLEncoded(b, beencode.Decode(b))
// 	want := "%02r%fd%fe%bf%fbt%d0%0f%cf%d9%8c%e0%a9%97%f8%08%9b%00%b2"
// 	assertAreEqual(T, got, want)
// }
