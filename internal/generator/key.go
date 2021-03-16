package generator

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
)

func NewDefaultKeyGenerator() (*DefaultKeyGenerator, error) {
	randomBytes := make([]byte, 4)
	rand.Read(randomBytes)
	str := hex.EncodeToString(randomBytes)
	result := strings.ToUpper(str)
	return &DefaultKeyGenerator{generated: result}, nil
}

type DefaultKeyGenerator struct {
	generated string
}

func (d *DefaultKeyGenerator) Key() string {
	return d.generated
}
