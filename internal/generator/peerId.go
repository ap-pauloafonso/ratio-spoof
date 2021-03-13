package generator

import (
	regen "github.com/zach-klippenstein/goregen"
)

type PeerIdGenerator interface {
	PeerId() string
}

type RegexPeerIdGenerator struct {
	generated string
}

func NewPeerIdGenerator(generatorCode, pattern string) (PeerIdGenerator, error) {
	result, err := regen.Generate(pattern)
	if err != nil {
		return nil, err
	}
	return &RegexPeerIdGenerator{generated: result}, nil
}

func (d *RegexPeerIdGenerator) PeerId() string {
	return d.generated
}
