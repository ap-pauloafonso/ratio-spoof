package emulation

import (
	"embed"
	"encoding/json"
	"io/ioutil"

	"github.com/ap-pauloafonso/ratio-spoof/internal/generator"
)

type ClientInfo struct {
	Name   string `json:"name"`
	PeerID struct {
		Generator string `json:"generator"`
		Regex     string `json:"regex"`
	} `json:"peerId"`
	Key struct {
		Generator string `json:"generator"`
		Regex     string `json:"regex"`
	} `json:"key"`
	Rounding struct {
		Generator string `json:"generator"`
		Regex     string `json:"regex"`
	} `json:"rounding"`
	Query   string            `json:"query"`
	Headers map[string]string `json:"headers"`
}

type Emulation struct {
	PeerIdGenerator  generator.PeerIdGenerator
	KeyGenerator     generator.KeyGenerator
	Query            string
	Name             string
	Headers          map[string]string
	RoudingGenerator generator.RoundingGenerator
}

func NewEmulation(code string) (*Emulation, error) {
	c, err := extractClient(code)
	if err != nil {
		return nil, err
	}

	peerG, err := generator.NewPeerIdGenerator(c.PeerID.Generator, c.PeerID.Regex)
	if err != nil {
		return nil, err
	}

	keyG, err := generator.NewKeyGenerator(c.Key.Generator)
	if err != nil {
		return nil, err
	}

	roudingG, err := generator.NewRoundingGenerator(c.Rounding.Generator)
	if err != nil {
		return nil, err
	}

	return &Emulation{PeerIdGenerator: peerG, KeyGenerator: keyG, RoudingGenerator: roudingG,
		Headers: c.Headers, Name: c.Name, Query: c.Query}, nil

}

//go:embed static
var staticFiles embed.FS

func extractClient(code string) (*ClientInfo, error) {

	f, err := staticFiles.Open("static/" + code + ".json")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	bytes, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	var client ClientInfo

	json.Unmarshal(bytes, &client)

	return &client, nil
}
