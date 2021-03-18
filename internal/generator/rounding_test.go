package generator

import "testing"

func TestDefaultRounding(t *testing.T) {
	r, _ := NewDefaultRoudingGenerator()

	d, u, l := r.Round(656497856, 46479878, 7879879, 1024)
	//same
	if d != 656497856 {
		t.Errorf("[download]got %v want %v", d, 656497856)
	}
	//16kb round
	if u != 46465024 {
		t.Errorf("[upload]got %v want %v", u, 46465024)
	}
	//piece size round
	if l != 7879680 {
		t.Errorf("[left]got %v want %v", l, 7879680)
	}
}
