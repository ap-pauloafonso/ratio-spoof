package ratiospoof

import (
	"io/ioutil"
	"testing"

	"github.com/ap-pauloafonso/ratio-spoof/beencode"
)

func assertAreEqual(t *testing.T, got, want interface{}) {
	t.Helper()
	if got != want {
		t.Errorf("\ngot: %v\n want: %v", got, want)
	}
}

// func TestStrSize2ByteSize(T *testing.T) {
// 	T.Run("100kb", func(t *testing.T) {
// 		got := strSize2ByteSize("100kb", 100)
// 		want := 102400
// 		assertAreEqual(t, got, want)
// 	})
// 	T.Run("1kb", func(t *testing.T) {
// 		got := strSize2ByteSize("1kb", 0)
// 		want := 1024
// 		assertAreEqual(t, got, want)
// 	})

// 	T.Run("1mb", func(t *testing.T) {
// 		got := strSize2ByteSize("1mb", 0)
// 		want := 1048576
// 		assertAreEqual(t, got, want)
// 	})
// 	T.Run("1gb", func(t *testing.T) {
// 		got := strSize2ByteSize("1gb", 0)
// 		want := 1073741824
// 		assertAreEqual(t, got, want)
// 	})
// 	T.Run("1.5gb", func(t *testing.T) {
// 		got := strSize2ByteSize("1.5gb", 0)
// 		want := 1610612736
// 		assertAreEqual(t, got, want)
// 	})
// 	T.Run("1tb", func(t *testing.T) {
// 		got := strSize2ByteSize("1tb", 0)
// 		want := 1099511627776
// 		assertAreEqual(t, got, want)
// 	})
// 	T.Run("1b", func(t *testing.T) {
// 		got := strSize2ByteSize("1b", 0)
// 		want := 1
// 		assertAreEqual(t, got, want)
// 	})
// 	T.Run("100%% of 10gb ", func(t *testing.T) {
// 		got := strSize2ByteSize("100%", 10737418240)
// 		want := 10737418240
// 		assertAreEqual(t, got, want)
// 	})
// 	T.Run("55%% of 900mb ", func(t *testing.T) {
// 		got := strSize2ByteSize("55%", 943718400)
// 		want := 519045120
// 		assertAreEqual(t, got, want)
// 	})
// 	T.Run("55%% of 900mb ", func(t *testing.T) {
// 		got := strSize2ByteSize("55%", 943718400)
// 		want := 519045120
// 		assertAreEqual(t, got, want)
// 	})
// }

// func TestHumanReadableSize(T *testing.T) {

// 	T.Run("#1", func(t *testing.T) {
// 		got := humanReadableSize(1536, true)
// 		want := "1.50KiB"

// 		assertAreEqual(t, got, want)
// 	})
// 	T.Run("#2", func(t *testing.T) {
// 		got := humanReadableSize(379040563, true)
// 		want := "1.50KiB"

// 		assertAreEqual(t, got, want)
// 	})
// 	T.Run("#3", func(t *testing.T) {
// 		got := humanReadableSize(6291456, true)
// 		want := "1.50KiB"

// 		assertAreEqual(t, got, want)
// 	})
// 	T.Run("#4", func(t *testing.T) {
// 		got := humanReadableSize(372749107, true)
// 		want := "1.50KiB"

// 		assertAreEqual(t, got, want)
// 	})
// 	T.Run("#5", func(t *testing.T) {
// 		got := humanReadableSize(10485760, true)
// 		want := "1.50KiB"

// 		assertAreEqual(t, got, want)
// 	})
// 	T.Run("#6", func(t *testing.T) {
// 		got := humanReadableSize(15728640, true)
// 		want := "1.50KiB"
// 		assertAreEqual(t, got, want)
// 	})
// 	T.Run("#7", func(t *testing.T) {
// 		got := humanReadableSize(363311923, true)
// 		want := "1.50KiB"
// 		assertAreEqual(t, got, want)
// 	})
// 	T.Run("#8", func(t *testing.T) {
// 		got := humanReadableSize(16777216, true)
// 		want := "1.50KiB"

// 		assertAreEqual(t, got, want)
// 	})
// 	T.Run("#9", func(t *testing.T) {
// 		got := humanReadableSize(379040563, true)
// 		want := "1.50KiB"

// 		assertAreEqual(t, got, want)
// 	})
// }

func TestClculateNextTotalSizeByte(T *testing.T) {

	got := calculateNextTotalSizeByte(100, 0, 512, 30, 87979879)
	want := 3074560

	assertAreEqual(T, got, want)
}

func TestUrlEncodeInfoHash(T *testing.T) {

	b, _ := ioutil.ReadFile("")
	got := extractInfoHashURLEncoded(b, beencode.Decode(b))
	want := "%60N%7d%1f%8b%3a%9bT%d5%fc%ad%d1%27%ab5%02%1c%fb%03%b0"
	assertAreEqual(T, got, want)
}
