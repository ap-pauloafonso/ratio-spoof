package generator

import (
	regen "github.com/zach-klippenstein/goregen"
)

type RegexPeerIdGenerator struct {
	generated string
}

func NewRegexPeerIdGenerator(pattern string) (*RegexPeerIdGenerator, error) {
	result, err := regen.Generate(pattern)
	if err != nil {
		return nil, err
	}
	return &RegexPeerIdGenerator{generated: result}, nil
}

func (d *RegexPeerIdGenerator) PeerId() string {
	return d.generated
}
