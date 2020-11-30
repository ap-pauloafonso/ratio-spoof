package qbittorrent

import (
	"strings"
	"testing"
)

func TestGenerateRandomPeerId(T *testing.T) {
	T.Run("PeerIds are different", func(t *testing.T) {
		keys := make(map[string]bool)
		for i := 0; i < 10; i++ {
			obj := NewQbitTorrent()
			key := obj.PeerID()
			t.Log(key)
			if _, ok := keys[key]; ok {
				t.Error("peerId must be random")
				break
			}
			keys[key] = true
		}

	})

}

func TestGenerateRandomKey(T *testing.T) {
	T.Run("Keys are different", func(t *testing.T) {
		keys := make(map[string]bool)
		for i := 0; i < 10; i++ {
			obj := NewQbitTorrent()
			key := obj.Key()
			t.Log(key)
			if _, ok := keys[key]; ok {
				t.Error("Keys must be random")
				break
			}
			keys[key] = true
		}
	})
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
