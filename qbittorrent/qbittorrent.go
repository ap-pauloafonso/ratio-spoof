package qbittorrent

import (
	"encoding/hex"
	"math/rand"
	"strings"
	"time"
)

const (
	letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_"
	name        = "qBittorrent v4.03"
	query       = "info_hash={infohash}&peer_id={peerid}&port={port}&uploaded={uploaded}&downloaded={downloaded}&left={left}&corrupt=0&key={key}&event={event}&numwant={numwant}&compact=1&no_peer_id=1&supportcrypto=1&redundant=0"
)

type TypeTest struct {
}

type qbitTorrent struct {
	name        string
	query       string
	dictHeaders map[string]string
	key         string
	peerID      string
}

func NewQbitTorrent() *qbitTorrent {
	return &qbitTorrent{
		name:        name,
		query:       query,
		dictHeaders: generateHeaders(),
		key:         generateKey(),
		peerID:      generatePeerID(),
	}
}

func (qb *qbitTorrent) Name() string {
	return qb.name
}
func (qb *qbitTorrent) PeerID() string {
	return qb.peerID
}

func (qb *qbitTorrent) Key() string {
	return qb.key
}

func (qb *qbitTorrent) Query() string {
	return query
}
func (qb *qbitTorrent) Headers() map[string]string {
	return qb.dictHeaders
}

func generateHeaders() map[string]string {
	return map[string]string{"User-Agent": "qBittorrent/4.0.3", "Accept-Encoding": "gzip"}
}

func generatePeerID() string {
	return "-qB4030-" + randStringBytes(12)
}

func randStringBytes(n int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}
func generateKey() string {
	randomBytes := make([]byte, 4)
	rand.Read(randomBytes)
	str := hex.EncodeToString(randomBytes)
	return strings.ToUpper(str)
}
