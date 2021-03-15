package generator

import "testing"

func TestDeaultKeyGenerator(t *testing.T) {
	t.Run("Key has 8 length", func(t *testing.T) {
		obj, _ := NewDefaultKeyGenerator()
		key := obj.Key()
		if len(key) != 8 {
			t.Error("Keys must have length of 8")
		}

	})
}
