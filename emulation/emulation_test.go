package emulation

import (
	"io/fs"
	"strings"
	"testing"
)

func TestNewEmulation(t *testing.T) {
	var counter int
	fs.WalkDir(staticFiles, ".", func(path string, d fs.DirEntry, err error) error {
		if counter > 1 {
			code := strings.TrimRight(strings.TrimLeft(path, "static/"), ".json")
			e, err := NewEmulation(code)
			if err != nil {
				t.Error("should not return error ")
			}

			peerId := e.PeerId()
			key := e.Key()

			d, u, l := e.Round(2*1024*1024*1024, 1024*1024*1024, 3*1024*1024*1024, 1024)

			if peerId == "" {
				t.Errorf("%s.json should be able to generate PeerId", code)
			}
			if key == "" {
				t.Errorf("%s.json should be able to generate Key", code)
			}
			if d <= 0 || u <= 0 || l <= 0 {
				t.Errorf("%s.json should be able to round candidates", code)
			}
		}
		counter++
		return nil
	})

}
func TestExtractClient(t *testing.T) {
	var counter int
	fs.WalkDir(staticFiles, ".", func(path string, d fs.DirEntry, err error) error {
		if counter > 1 {
			code := strings.TrimRight(strings.TrimLeft(path, "static/"), ".json")
			c, e := extractClient(code)
			if e != nil || err != nil {
				t.Error("should not return error")
			}

			if c.Key.Generator == "" && c.Key.Regex == "" {
				t.Errorf("%s.json should have key generator properties", code)
			}
			if c.PeerID.Generator == "" && c.PeerID.Regex == "" {
				t.Errorf("%s.json should have PeerId generator properties", code)
			}

			if c.Rounding.Generator == "" && c.Rounding.Regex == "" {
				t.Errorf("%s.json should have rouding generator properties", code)
			}

			if c.Name == "" {
				t.Errorf("%s.json should have a name", code)
			}
			if c.Query == "" {
				t.Errorf("%s.json should have a query", code)
			}
			if len(c.Headers) == 0 {
				t.Errorf("%s.json should have headers", code)
			}
		}
		counter++
		return nil
	})

}
