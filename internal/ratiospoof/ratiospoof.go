package ratiospoof

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ap-pauloafonso/ratio-spoof/internal/bencode"
	"github.com/ap-pauloafonso/ratio-spoof/internal/emulation"
	"github.com/ap-pauloafonso/ratio-spoof/internal/input"
	"github.com/ap-pauloafonso/ratio-spoof/internal/tracker"
	"github.com/gammazero/deque"
)

const (
	maxAnnounceHistory = 10
)

type RatioSpoof struct {
	mutex                           *sync.Mutex
	TorrentInfo                     *bencode.TorrentInfo
	Input                           *input.InputParsed
	Tracker                         *tracker.HttpTracker
	BitTorrentClient                *emulation.Emulation
	AnnounceInterval                int
	EstimatedTimeToAnnounce         time.Time
	EstimatedTimeToAnnounceUpdateCh chan int
	NumWant                         int
	Seeders                         int
	Leechers                        int
	AnnounceCount                   int
	Status                          string
	AnnounceHistory                 announceHistory
	StopPrintCH                     chan interface{}
}

type AnnounceEntry struct {
	Count             int
	Downloaded        int
	PercentDownloaded float32
	Uploaded          int
	Left              int
}

type announceHistory struct {
	deque.Deque
}

func NewRatioSpoofState(input input.InputArgs) (*RatioSpoof, error) {
	EstimatedTimeToAnnounceUpdateCh := make(chan int)
	stopPrintCh := make(chan interface{})
	dat, err := ioutil.ReadFile(input.TorrentPath)
	if err != nil {
		return nil, err
	}

	client, err := emulation.NewEmulation(input.Client)
	if err != nil {
		return nil, errors.New("Error building the emulated client with the code")
	}

	torrentInfo, err := bencode.TorrentDictParse(dat)
	if err != nil {
		return nil, errors.New("failed to parse the torrent file")
	}

	httpTracker, err := tracker.NewHttpTracker(torrentInfo)
	if err != nil {
		return nil, err
	}

	inputParsed, err := input.ParseInput(torrentInfo)
	if err != nil {
		return nil, err
	}

	return &RatioSpoof{
		BitTorrentClient:                client,
		TorrentInfo:                     torrentInfo,
		Tracker:                         httpTracker,
		Input:                           inputParsed,
		NumWant:                         200,
		Status:                          "started",
		mutex:                           &sync.Mutex{},
		StopPrintCH:                     stopPrintCh,
		EstimatedTimeToAnnounceUpdateCh: EstimatedTimeToAnnounceUpdateCh,
	}, nil
}

func (A *announceHistory) pushValueHistory(value AnnounceEntry) {
	if A.Len() >= maxAnnounceHistory {
		A.PopFront()
	}
	A.PushBack(value)
}

func (R *RatioSpoof) gracefullyExit() {
	fmt.Printf("\nGracefully exiting...\n")
	R.Status = "stopped"
	R.NumWant = 0
	R.fireAnnounce(false)
	fmt.Printf("Gracefully exited successfully.\n")

}

func (R *RatioSpoof) Run() {
	rand.Seed(time.Now().UnixNano())
	sigCh := make(chan os.Signal)

	signal.Notify(sigCh, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	R.firstAnnounce()
	go R.updateEstimatedTimeToAnnounceListener()
	go func() {
		for {
			R.generateNextAnnounce()
			time.Sleep(time.Duration(R.AnnounceInterval) * time.Second)
			R.fireAnnounce(true)
		}
	}()
	<-sigCh
	R.StopPrintCH <- "exit print"
	R.gracefullyExit()
}
func (R *RatioSpoof) firstAnnounce() {
	R.addAnnounce(R.Input.InitialDownloaded, R.Input.InitialUploaded, calculateBytesLeft(R.Input.InitialDownloaded, R.TorrentInfo.TotalSize), (float32(R.Input.InitialDownloaded)/float32(R.TorrentInfo.TotalSize))*100)
	R.fireAnnounce(false)
}

func (R *RatioSpoof) updateInterval(interval int) {
	if interval > 0 {
		R.AnnounceInterval = interval
	} else {
		R.AnnounceInterval = 1800
	}
	R.updateEstimatedTimeToAnnounce(R.AnnounceInterval)
}

func (R *RatioSpoof) updateEstimatedTimeToAnnounce(interval int) {
	R.mutex.Lock()
	defer R.mutex.Unlock()
	R.EstimatedTimeToAnnounce = time.Now().Add(time.Duration(interval) * time.Second)
}

func (R *RatioSpoof) updateEstimatedTimeToAnnounceListener() {
	for {
		interval := <-R.EstimatedTimeToAnnounceUpdateCh
		R.updateEstimatedTimeToAnnounce(interval)
	}
}

func (R *RatioSpoof) updateSeedersAndLeechers(resp tracker.TrackerResponse) {
	R.Seeders = resp.Seeders
	R.Leechers = resp.Leechers
}
func (R *RatioSpoof) addAnnounce(currentDownloaded, currentUploaded, currentLeft int, percentDownloaded float32) {
	R.AnnounceCount++
	R.AnnounceHistory.pushValueHistory(AnnounceEntry{Count: R.AnnounceCount, Downloaded: currentDownloaded, Uploaded: currentUploaded, Left: currentLeft, PercentDownloaded: percentDownloaded})
}
func (R *RatioSpoof) fireAnnounce(retry bool) error {
	lastAnnounce := R.AnnounceHistory.Back().(AnnounceEntry)
	replacer := strings.NewReplacer("{infohash}", R.TorrentInfo.InfoHashURLEncoded,
		"{port}", fmt.Sprint(R.Input.Port),
		"{peerid}", R.BitTorrentClient.PeerIdGenerator.PeerId(),
		"{uploaded}", fmt.Sprint(lastAnnounce.Uploaded),
		"{downloaded}", fmt.Sprint(lastAnnounce.Downloaded),
		"{left}", fmt.Sprint(lastAnnounce.Left),
		"{key}", R.BitTorrentClient.KeyGenerator.Key(),
		"{event}", R.Status,
		"{numwant}", fmt.Sprint(R.NumWant))
	query := replacer.Replace(R.BitTorrentClient.Query)
	trackerResp, err := R.Tracker.Announce(query, R.BitTorrentClient.Headers, retry, R.EstimatedTimeToAnnounceUpdateCh)
	if err != nil {
		log.Fatalf("failed to reach the tracker:\n%s ", err.Error())
	}

	if trackerResp != nil {
		R.updateSeedersAndLeechers(*trackerResp)
		R.updateInterval(trackerResp.Interval)
	}
	return nil
}
func (R *RatioSpoof) generateNextAnnounce() {
	lastAnnounce := R.AnnounceHistory.Back().(AnnounceEntry)
	currentDownloaded := lastAnnounce.Downloaded
	var downloadCandidate int

	if currentDownloaded < R.TorrentInfo.TotalSize {
		downloadCandidate = calculateNextTotalSizeByte(R.Input.DownloadSpeed, currentDownloaded, R.TorrentInfo.PieceSize, R.AnnounceInterval, R.TorrentInfo.TotalSize)
	} else {
		downloadCandidate = R.TorrentInfo.TotalSize
	}

	currentUploaded := lastAnnounce.Uploaded
	uploadCandidate := calculateNextTotalSizeByte(R.Input.UploadSpeed, currentUploaded, R.TorrentInfo.PieceSize, R.AnnounceInterval, 0)

	leftCandidate := calculateBytesLeft(downloadCandidate, R.TorrentInfo.TotalSize)

	d, u, l := R.BitTorrentClient.RoudingGenerator.NextAmountReport(downloadCandidate, uploadCandidate, leftCandidate, R.TorrentInfo.PieceSize)

	R.addAnnounce(d, u, l, (float32(d)/float32(R.TorrentInfo.TotalSize))*100)
}

func calculateNextTotalSizeByte(speedBytePerSecond, currentByte, pieceSizeByte, seconds, limitTotalBytes int) int {
	if speedBytePerSecond == 0 {
		return currentByte
	}
	totalCandidate := currentByte + (speedBytePerSecond * seconds)
	randomPieces := rand.Intn(10-1) + 1
	totalCandidate = totalCandidate + (pieceSizeByte * randomPieces)

	if limitTotalBytes != 0 && totalCandidate > limitTotalBytes {
		return limitTotalBytes
	}
	return totalCandidate
}

func calculateBytesLeft(currentBytes, totalBytes int) int {
	return totalBytes - currentBytes
}
