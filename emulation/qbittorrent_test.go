package emulation

import (
	"strings"
	"testing"
)

func TestGenerateRandomKey(T *testing.T) {
	T.Run("Key has 8 length", func(t *testing.T) {
		obj := NewQbitTorrent()
		key := obj.Key()
		if len(key) != 8 {
			t.Error("Keys must have length of 8")
		}

	})
	T.Run("Key must be uppercase", func(t *testing.T) {
		obj := NewQbitTorrent()
		key := obj.Key()
		if strings.ToUpper(key) != key {
			t.Error("Keys must be uppercase")
		}

	})

}
