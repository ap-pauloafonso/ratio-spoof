package ratiospoof

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ap-pauloafonso/ratio-spoof/beencode"
	"github.com/gammazero/deque"
)

const (
	maxAnnounceHistory = 10
)

var validInitialSufixes = [...]string{"%", "b", "kb", "mb", "gb", "tb"}
var validSpeedSufixes = [...]string{"kbps", "mbps"}

type ratioSpoofState struct {
	mutex                *sync.Mutex
	httpClient           HttpClient
	torrentInfo          *beencode.TorrentInfo
	input                *inputParsed
	trackerState         *httpTracker
	bitTorrentClient     TorrentClientEmulation
	currentAnnounceTimer int
	announceInterval     int
	numWant              int
	seeders              int
	leechers             int
	announceCount        int
	status               string
	announceHistory      announceHistory
	lastAnounceRequest   string
	lastTackerResponse   string
	retryAttempt         int
}
type httpTracker struct {
	urls []string
}

func newHttpTracker(torrentInfo *beencode.TorrentInfo) (*httpTracker, error) {

	var result []string
	for _, url := range torrentInfo.TrackerInfo.Urls {
		if strings.HasPrefix(url, "http") {
			result = append(result, url)
		}
	}
	if len(result) == 0 {
		return nil, errors.New("No tcp/http tracker url announce found")
	}
	return &httpTracker{urls: torrentInfo.TrackerInfo.Urls}, nil
}

func (T *httpTracker) SwapFirst(currentIdx int) {
	aux := T.urls[0]
	T.urls[0] = T.urls[currentIdx]
	T.urls[currentIdx] = aux
}

type InputArgs struct {
	TorrentPath       string
	InitialDownloaded string
	DownloadSpeed     string
	InitialUploaded   string
	UploadSpeed       string
	Port              int
	Debug             bool
}

type inputParsed struct {
	torrentPath       string
	initialDownloaded int
	downloadSpeed     int
	initialUploaded   int
	uploadSpeed       int
	port              int
	debug             bool
}

func (I *InputArgs) parseInput(torrentInfo *beencode.TorrentInfo) (*inputParsed, error) {
	downloaded, err := extractInputInitialByteCount(I.InitialDownloaded, torrentInfo.TotalSize, true)
	if err != nil {
		return nil, err
	}
	uploaded, err := extractInputInitialByteCount(I.InitialUploaded, torrentInfo.TotalSize, false)
	if err != nil {
		return nil, err
	}
	downloadSpeed, err := extractInputByteSpeed(I.DownloadSpeed)
	if err != nil {
		return nil, err
	}
	uploadSpeed, err := extractInputByteSpeed(I.UploadSpeed)
	if err != nil {
		return nil, err
	}

	if I.Port < 1 || I.Port > 65535 {
		return nil, errors.New("port number must be between 1 and 65535")
	}

	return &inputParsed{initialDownloaded: downloaded,
		downloadSpeed:   downloadSpeed,
		initialUploaded: uploaded,
		uploadSpeed:     uploadSpeed,
		debug:           I.Debug,
		port:            I.Port,
	}, nil
}

func NewRatioSPoofState(input InputArgs, torrentClient TorrentClientEmulation, httpclient HttpClient) (*ratioSpoofState, error) {

	dat, err := ioutil.ReadFile(input.TorrentPath)
	if err != nil {
		return nil, err
	}

	torrentInfo, err := beencode.TorrentDictParse(dat)
	if err != nil {
		panic(err)
	}

	httpTracker, err := newHttpTracker(torrentInfo)
	if err != nil {
		panic(err)
	}

	inputParsed, err := input.parseInput(torrentInfo)
	if err != nil {
		panic(err)
	}

	return &ratioSpoofState{
		bitTorrentClient: torrentClient,
		httpClient:       httpclient,
		torrentInfo:      torrentInfo,
		trackerState:     httpTracker,
		input:            inputParsed,
		numWant:          200,
		status:           "started",
		mutex:            &sync.Mutex{},
	}, nil
}

func checkSpeedSufix(input string) (valid bool, suffix string) {
	for _, v := range validSpeedSufixes {

		if strings.HasSuffix(strings.ToLower(input), v) {
			return true, input[len(input)-4:]
		}
	}
	return false, ""
}

func extractInputInitialByteCount(initialSizeInput string, totalBytes int, errorIfHigher bool) (int, error) {
	byteCount := strSize2ByteSize(initialSizeInput, totalBytes)
	if errorIfHigher && byteCount > totalBytes {
		return 0, errors.New("initial downloaded can not be higher than the torrent size")
	}
	if byteCount < 0 {
		return 0, errors.New("initial value can not be negative")
	}
	return byteCount, nil
}
func extractInputByteSpeed(initialSpeedInput string) (int, error) {
	ok, suffix := checkSpeedSufix(initialSpeedInput)
	if !ok {
		return 0, fmt.Errorf("speed must be in %v", validSpeedSufixes)
	}
	number, _ := strconv.ParseFloat(initialSpeedInput[:len(initialSpeedInput)-4], 64)
	if number < 0 {
		return 0, errors.New("speed can not be negative")
	}

	if suffix == "kbps" {
		number *= 1024
	} else {
		number = number * 1024 * 1024
	}
	ret := int(number)
	return ret, nil
}

type trackerResponse struct {
	minInterval int
	interval    int
	seeders     int
	leechers    int
}

type TorrentClientEmulation interface {
	PeerID() string
	Key() string
	Query() string
	Name() string
	Headers() map[string]string
	NextAmountReport(DownloadCandidateNextAmount, UploadCandidateNextAmount, leftCandidateNextAmount, pieceSize int) (downloaded, uploaded, left int)
}
type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type announceEntry struct {
	count             int
	downloaded        int
	percentDownloaded float32
	uploaded          int
	left              int
}

type announceHistory struct {
	deque.Deque
}

func (A *announceHistory) pushValueHistory(value announceEntry) {
	if A.Len() >= maxAnnounceHistory {
		A.PopFront()
	}
	A.PushBack(value)
}

func (R *ratioSpoofState) gracefullyExit() {
	fmt.Printf("\nGracefully exiting...\n")
	R.status = "stopped"
	R.numWant = 0
	R.fireAnnounce(false)
}

func (R *ratioSpoofState) Run() {
	rand.Seed(time.Now().UnixNano())
	sigCh := make(chan os.Signal)
	stopPrintCh := make(chan string)

	signal.Notify(sigCh, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	R.firstAnnounce()
	go R.decreaseTimer()
	go R.PrintState(stopPrintCh)
	go func() {
		for {
			R.generateNextAnnounce()
			time.Sleep(time.Duration(R.announceInterval) * time.Second)
			R.fireAnnounce(true)
		}
	}()
	<-sigCh
	stopPrintCh <- "exit print"
	R.gracefullyExit()
}
func (R *ratioSpoofState) firstAnnounce() {
	println("Trying to connect to the tracker...")
	R.addAnnounce(R.input.initialDownloaded, R.input.initialUploaded, calculateBytesLeft(R.input.initialDownloaded, R.torrentInfo.TotalSize), (float32(R.input.initialDownloaded)/float32(R.torrentInfo.TotalSize))*100)
	R.fireAnnounce(false)
}

func (R *ratioSpoofState) updateInterval(resp trackerResponse) {
	if resp.minInterval > 0 {
		R.announceInterval = resp.minInterval
	} else {
		R.announceInterval = resp.interval
	}
}

func (R *ratioSpoofState) updateSeedersAndLeechers(resp trackerResponse) {
	R.seeders = resp.seeders
	R.leechers = resp.leechers
}
func (R *ratioSpoofState) addAnnounce(currentDownloaded, currentUploaded, currentLeft int, percentDownloaded float32) {
	R.announceCount++
	R.announceHistory.pushValueHistory(announceEntry{count: R.announceCount, downloaded: currentDownloaded, uploaded: currentUploaded, left: currentLeft, percentDownloaded: percentDownloaded})
}
func (R *ratioSpoofState) fireAnnounce(retry bool) {
	lastAnnounce := R.announceHistory.Back().(announceEntry)
	replacer := strings.NewReplacer("{infohash}", R.torrentInfo.InfoHashURLEncoded,
		"{port}", fmt.Sprint(R.input.port),
		"{peerid}", R.bitTorrentClient.PeerID(),
		"{uploaded}", fmt.Sprint(lastAnnounce.uploaded),
		"{downloaded}", fmt.Sprint(lastAnnounce.downloaded),
		"{left}", fmt.Sprint(lastAnnounce.left),
		"{key}", R.bitTorrentClient.Key(),
		"{event}", R.status,
		"{numwant}", fmt.Sprint(R.numWant))
	query := replacer.Replace(R.bitTorrentClient.Query())

	var trackerResp *trackerResponse
	if retry {
		retryDelay := 30 * time.Second
		for {
			exit := false
			func() {
				defer func() {
					if err := recover(); err != nil {
						R.changeCurrentTimer(int(retryDelay.Seconds()))
						R.retryAttempt++
						time.Sleep(retryDelay)
						retryDelay *= 2
						if retryDelay.Seconds() > 900 {
							retryDelay = 900
						}
					}
				}()
				trackerResp = R.tryMakeRequest(query)
				exit = true
			}()
			if exit {
				break
			}
		}

	} else {
		trackerResp = R.tryMakeRequest(query)
	}
	R.retryAttempt = 0
	if trackerResp != nil {
		R.updateSeedersAndLeechers(*trackerResp)
		R.updateInterval(*trackerResp)
	}
}
func (R *ratioSpoofState) generateNextAnnounce() {
	R.changeCurrentTimer(R.announceInterval)
	lastAnnounce := R.announceHistory.Back().(announceEntry)
	currentDownloaded := lastAnnounce.downloaded
	var downloadCandidate int

	if currentDownloaded < R.torrentInfo.TotalSize {
		downloadCandidate = calculateNextTotalSizeByte(R.input.downloadSpeed, currentDownloaded, R.torrentInfo.PieceSize, R.currentAnnounceTimer, R.torrentInfo.TotalSize)
	} else {
		downloadCandidate = R.torrentInfo.TotalSize
	}

	currentUploaded := lastAnnounce.uploaded
	uploadCandidate := calculateNextTotalSizeByte(R.input.uploadSpeed, currentUploaded, R.torrentInfo.PieceSize, R.currentAnnounceTimer, 0)

	leftCandidate := calculateBytesLeft(downloadCandidate, R.torrentInfo.TotalSize)

	d, u, l := R.bitTorrentClient.NextAmountReport(downloadCandidate, uploadCandidate, leftCandidate, R.torrentInfo.PieceSize)

	R.addAnnounce(d, u, l, (float32(d)/float32(R.torrentInfo.TotalSize))*100)
}
func (R *ratioSpoofState) decreaseTimer() {
	for {
		time.Sleep(1 * time.Second)
		R.mutex.Lock()
		if R.currentAnnounceTimer > 0 {
			R.currentAnnounceTimer--
		}
		R.mutex.Unlock()
	}
}

func (R *ratioSpoofState) changeCurrentTimer(newAnnounceRate int) {
	R.mutex.Lock()
	R.currentAnnounceTimer = newAnnounceRate
	R.mutex.Unlock()
}

func (R *ratioSpoofState) tryMakeRequest(query string) *trackerResponse {
	for idx, url := range R.torrentInfo.TrackerInfo.Urls {
		completeURL := url + "?" + strings.TrimLeft(query, "?")
		R.lastAnounceRequest = completeURL
		req, _ := http.NewRequest("GET", completeURL, nil)
		for header, value := range R.bitTorrentClient.Headers() {
			req.Header.Add(header, value)
		}
		resp, err := R.httpClient.Do(req)
		if err == nil {
			if resp.StatusCode == http.StatusOK {
				bytesR, _ := ioutil.ReadAll(resp.Body)
				if len(bytesR) == 0 {
					return nil
				}
				mimeType := http.DetectContentType(bytesR)
				if mimeType == "application/x-gzip" {
					gzipReader, _ := gzip.NewReader(bytes.NewReader(bytesR))
					bytesR, _ = ioutil.ReadAll(gzipReader)
					gzipReader.Close()
				}
				R.lastTackerResponse = string(bytesR)
				decodedResp := beencode.Decode(bytesR)
				if idx != 0 {
					R.trackerState.SwapFirst(idx)
				}
				ret := extractTrackerResponse(decodedResp)
				return &ret
			}
			resp.Body.Close()
		}
	}
	panic("Connection error with the tracker")

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

func extractTrackerResponse(datatrackerResponse map[string]interface{}) trackerResponse {
	var result trackerResponse
	if v, ok := datatrackerResponse["failure reason"].(string); ok && len(v) > 0 {
		panic(errors.New(v))
	}
	result.minInterval, _ = datatrackerResponse["min interval"].(int)
	result.interval, _ = datatrackerResponse["interval"].(int)
	result.seeders, _ = datatrackerResponse["complete"].(int)
	result.leechers, _ = datatrackerResponse["incomplete"].(int)
	return result

}
func calculateBytesLeft(currentBytes, totalBytes int) int {
	return totalBytes - currentBytes
}

func strSize2ByteSize(input string, totalSize int) int {
	lowerInput := strings.ToLower(input)

	parseStrNumberFn := func(strWithSufix string, sufixLength, n int) int {
		v, _ := strconv.ParseFloat(strWithSufix[:len(lowerInput)-sufixLength], 64)
		result := v * math.Pow(1024, float64(n))
		return int(result)
	}
	switch {
	case strings.HasSuffix(lowerInput, "kb"):
		{
			return parseStrNumberFn(lowerInput, 2, 1)
		}
	case strings.HasSuffix(lowerInput, "mb"):
		{
			return parseStrNumberFn(lowerInput, 2, 2)
		}
	case strings.HasSuffix(lowerInput, "gb"):
		{
			return parseStrNumberFn(lowerInput, 2, 3)
		}
	case strings.HasSuffix(lowerInput, "tb"):
		{
			return parseStrNumberFn(lowerInput, 2, 4)
		}
	case strings.HasSuffix(lowerInput, "b"):
		{
			return parseStrNumberFn(lowerInput, 1, 0)
		}
	case strings.HasSuffix(lowerInput, "%"):
		{
			v, _ := strconv.ParseFloat(lowerInput[:len(lowerInput)-1], 64)
			if v < 0 || v > 100 {
				panic("percent value must be in (0-100)")
			}
			result := int(float64(v/100) * float64(totalSize))

			return result
		}

	default:
		panic("Size not found")
	}
}
