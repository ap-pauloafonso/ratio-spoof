package ratiospoof

import (
	"testing"
)

func assertAreEqual(t *testing.T, got, want interface{}) {
	t.Helper()
	if got != want {
		t.Errorf("\ngot : %v\nwant: %v", got, want)
	}
}

func TestClculateNextTotalSizeByte(T *testing.T) {

	got := calculateNextTotalSizeByte(100*1024, 0, 512, 30, 87979879)
	want := 3075072

	assertAreEqual(T, got, want)
}

// func TestUrlEncodeInfoHash(T *testing.T) {

// 	b, _ := ioutil.ReadFile("")
// 	got := extractInfoHashURLEncoded(b, bencode.Decode(b))
// 	want := "%60N%7d%1f%8b%3a%9bT%d5%fc%ad%d1%27%ab5%02%1c%fb%03%b0"
// 	assertAreEqual(T, got, want)
// }

// func TestUrlEncodeInfoHash2(T *testing.T) {

// 	b, _ := ioutil.ReadFile("")
// 	got := extractInfoHashURLEncoded(b, bencode.Decode(b))
// 	want := "%02r%fd%fe%bf%fbt%d0%0f%cf%d9%8c%e0%a9%97%f8%08%9b%00%b2"
// 	assertAreEqual(T, got, want)
// }
