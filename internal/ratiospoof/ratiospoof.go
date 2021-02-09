package ratiospoof

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ap-pauloafonso/ratio-spoof/internal/bencode"
	"github.com/ap-pauloafonso/ratio-spoof/internal/input"
	"github.com/ap-pauloafonso/ratio-spoof/internal/tracker"
	"github.com/gammazero/deque"
)

const (
	maxAnnounceHistory = 10
)

type RatioSpoof struct {
	mutex                *sync.Mutex
	TorrentInfo          *bencode.TorrentInfo
	Input                *input.InputParsed
	Tracker              *tracker.HttpTracker
	BitTorrentClient     TorrentClientEmulation
	CurrentAnnounceTimer int
	AnnounceInterval     int
	NumWant              int
	Seeders              int
	Leechers             int
	AnnounceCount        int
	Status               string
	AnnounceHistory      announceHistory
	timerUpdateCh        chan int
	StopPrintCH          chan interface{}
}

type TorrentClientEmulation interface {
	PeerID() string
	Key() string
	Query() string
	Name() string
	Headers() map[string]string
	NextAmountReport(DownloadCandidateNextAmount, UploadCandidateNextAmount, leftCandidateNextAmount, pieceSize int) (downloaded, uploaded, left int)
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

func NewRatioSpoofState(input input.InputArgs, torrentClient TorrentClientEmulation) (*RatioSpoof, error) {
	changeTimerCh := make(chan int)
	stopPrintCh := make(chan interface{})
	dat, err := ioutil.ReadFile(input.TorrentPath)
	if err != nil {
		return nil, err
	}

	torrentInfo, err := bencode.TorrentDictParse(dat)
	if err != nil {
		panic(err)
	}

	httpTracker, err := tracker.NewHttpTracker(torrentInfo, changeTimerCh)
	if err != nil {
		panic(err)
	}

	inputParsed, err := input.ParseInput(torrentInfo)
	if err != nil {
		panic(err)
	}

	return &RatioSpoof{
		BitTorrentClient: torrentClient,
		TorrentInfo:      torrentInfo,
		Tracker:          httpTracker,
		Input:            inputParsed,
		NumWant:          200,
		Status:           "started",
		mutex:            &sync.Mutex{},
		timerUpdateCh:    changeTimerCh,
		StopPrintCH:      stopPrintCh,
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
}

func (R *RatioSpoof) Run() {
	rand.Seed(time.Now().UnixNano())
	sigCh := make(chan os.Signal)

	signal.Notify(sigCh, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	R.firstAnnounce()
	go R.decreaseTimer()
	go R.updateTimer()
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

func (R *RatioSpoof) updateInterval(resp tracker.TrackerResponse) {
	if resp.Interval > 0 {
		R.AnnounceInterval = resp.Interval
	} else {
		R.AnnounceInterval = 1800
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
func (R *RatioSpoof) fireAnnounce(retry bool) {
	lastAnnounce := R.AnnounceHistory.Back().(AnnounceEntry)
	replacer := strings.NewReplacer("{infohash}", R.TorrentInfo.InfoHashURLEncoded,
		"{port}", fmt.Sprint(R.Input.Port),
		"{peerid}", R.BitTorrentClient.PeerID(),
		"{uploaded}", fmt.Sprint(lastAnnounce.Uploaded),
		"{downloaded}", fmt.Sprint(lastAnnounce.Downloaded),
		"{left}", fmt.Sprint(lastAnnounce.Left),
		"{key}", R.BitTorrentClient.Key(),
		"{event}", R.Status,
		"{numwant}", fmt.Sprint(R.NumWant))
	query := replacer.Replace(R.BitTorrentClient.Query())
	trackerResp := R.Tracker.Announce(query, R.BitTorrentClient.Headers(), retry, R.timerUpdateCh)

	if trackerResp != nil {
		R.updateSeedersAndLeechers(*trackerResp)
		R.updateInterval(*trackerResp)
	}
}
func (R *RatioSpoof) generateNextAnnounce() {
	R.timerUpdateCh <- R.AnnounceInterval
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

	d, u, l := R.BitTorrentClient.NextAmountReport(downloadCandidate, uploadCandidate, leftCandidate, R.TorrentInfo.PieceSize)

	R.addAnnounce(d, u, l, (float32(d)/float32(R.TorrentInfo.TotalSize))*100)
}
func (R *RatioSpoof) decreaseTimer() {
	for {
		time.Sleep(1 * time.Second)
		R.mutex.Lock()
		if R.CurrentAnnounceTimer > 0 {
			R.CurrentAnnounceTimer--
		}
		R.mutex.Unlock()
	}
}

func (R *RatioSpoof) updateTimer() {
	for {
		newValue := <-R.timerUpdateCh
		R.mutex.Lock()
		R.CurrentAnnounceTimer = newValue
		R.mutex.Unlock()
	}
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
