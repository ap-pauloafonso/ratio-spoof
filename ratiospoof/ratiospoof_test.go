package ratiospoof

import (
	"testing"
)

func TestCalculateNextTotalSizeByte(t *testing.T) {
	randomPieces := 8
	got := calculateNextTotalSizeByte(100*1024, 0, 512, 30, 87979879, randomPieces)
	want := 3076096

	if got != want {
		t.Errorf("\ngot : %v\nwant: %v", got, want)
	}
}
