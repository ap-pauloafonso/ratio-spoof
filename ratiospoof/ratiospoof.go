package ratiospoof

import (
	"bytes"
	"compress/gzip"
	"crypto/sha1"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"regexp"
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
	torrentInfo          *torrentInfo
	input                *inputParsed
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

type torrentInfo struct {
	name               string
	pieceSize          int
	totalSize          int
	trackerInfo        trackerInfo
	infoHashURLEncoded string
}

func extractTorrentInfo(torrentPath string) (*torrentInfo, error) {
	dat, err := ioutil.ReadFile(torrentPath)
	if err != nil {
		return nil, err
	}
	torrentMap := beencode.Decode(dat)
	return &torrentInfo{
		name:               torrentMap["info"].(map[string]interface{})["name"].(string),
		pieceSize:          torrentMap["info"].(map[string]interface{})["piece length"].(int),
		totalSize:          extractTotalSize(torrentMap),
		trackerInfo:        extractTrackerInfo(torrentMap),
		infoHashURLEncoded: extractInfoHashURLEncoded(dat, torrentMap),
	}, nil
}
func (I *InputArgs) parseInput(torrentInfo *torrentInfo) (*inputParsed, error) {
	downloaded, err := extractInputInitialByteCount(I.InitialDownloaded, torrentInfo.totalSize, true)
	if err != nil {
		return nil, err
	}
	uploaded, err := extractInputInitialByteCount(I.InitialUploaded, torrentInfo.totalSize, false)
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

	torrentInfo, err := extractTorrentInfo(input.TorrentPath)
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

type trackerInfo struct {
	main string
	urls []string
}

func (T *trackerInfo) SwapFirst(currentIdx int) {
	aux := T.urls[0]
	T.urls[0] = T.urls[currentIdx]
	T.urls[currentIdx] = aux
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
	R.addAnnounce(R.input.initialDownloaded, R.input.initialUploaded, calculateBytesLeft(R.input.initialDownloaded, R.torrentInfo.totalSize), (float32(R.input.initialDownloaded)/float32(R.torrentInfo.totalSize))*100)
	R.fireAnnounce(false)
}

func (R *ratioSpoofState) updateInterval(resp trackerResponse) {
	if resp.interval > 0 {
		R.announceInterval = resp.interval
	} else {
		R.announceInterval = 1800
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
	replacer := strings.NewReplacer("{infohash}", R.torrentInfo.infoHashURLEncoded,
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

	if currentDownloaded < R.torrentInfo.totalSize {
		downloadCandidate = calculateNextTotalSizeByte(R.input.downloadSpeed, currentDownloaded, R.torrentInfo.pieceSize, R.currentAnnounceTimer, R.torrentInfo.totalSize)
	} else {
		downloadCandidate = R.torrentInfo.totalSize
	}

	currentUploaded := lastAnnounce.uploaded
	uploadCandidate := calculateNextTotalSizeByte(R.input.uploadSpeed, currentUploaded, R.torrentInfo.pieceSize, R.currentAnnounceTimer, 0)

	leftCandidate := calculateBytesLeft(downloadCandidate, R.torrentInfo.totalSize)

	d, u, l := R.bitTorrentClient.NextAmountReport(downloadCandidate, uploadCandidate, leftCandidate, R.torrentInfo.pieceSize)

	R.addAnnounce(d, u, l, (float32(d)/float32(R.torrentInfo.totalSize))*100)
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
	for idx, baseUrl := range R.torrentInfo.trackerInfo.urls {
		completeURL := buildFullUrl(baseUrl, query)
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
					R.torrentInfo.trackerInfo.SwapFirst(idx)
				}
				ret := extractTrackerResponse(decodedResp)
				return &ret
			}
			resp.Body.Close()
		}
	}
	panic("Connection error with the tracker")

}

func buildFullUrl(baseurl, query string) string {
	if len(strings.Split(baseurl, "?")) > 1 {
		return baseurl + "&" + strings.TrimLeft(query, "&")
	}
	return baseurl + "?" + strings.TrimLeft(query, "?")
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

func extractInfoHashURLEncoded(rawData []byte, torrentData map[string]interface{}) string {
	byteOffsets := torrentData["info"].(map[string]interface{})["byte_offsets"].([]int)
	h := sha1.New()
	h.Write([]byte(rawData[byteOffsets[0]:byteOffsets[1]]))
	ret := h.Sum(nil)
	var buf bytes.Buffer
	re := regexp.MustCompile(`[a-zA-Z0-9\.\-\_\~]`)
	for _, b := range ret {
		if re.Match([]byte{b}) {
			buf.WriteByte(b)
		} else {
			buf.WriteString(fmt.Sprintf("%%%02x", b))
		}
	}
	return buf.String()

}
func extractTotalSize(torrentData map[string]interface{}) int {
	if value, ok := torrentData["info"].(map[string]interface{})["length"]; ok {
		return value.(int)
	}

	var total int

	for _, file := range torrentData["info"].(map[string]interface{})["files"].([]interface{}) {
		total += file.(map[string]interface{})["length"].(int)
	}
	return total
}

func extractTrackerInfo(torrentData map[string]interface{}) trackerInfo {
	uniqueUrls := make(map[string]int)
	currentCount := 0
	if main, ok := torrentData["announce"]; ok && strings.HasPrefix(main.(string), "http") {
		if _, found := uniqueUrls[main.(string)]; !found {
			uniqueUrls[main.(string)] = currentCount
			currentCount++
		}
	}
	if list, ok := torrentData["announce-list"]; ok {
		for _, innerList := range list.([]interface{}) {
			for _, item := range innerList.([]interface{}) {
				if _, found := uniqueUrls[item.(string)]; !found && strings.HasPrefix(item.(string), "http") {
					uniqueUrls[item.(string)] = currentCount
					currentCount++
				}
			}
		}

	}
	trackerInfo := trackerInfo{urls: make([]string, len(uniqueUrls))}
	for key, value := range uniqueUrls {
		trackerInfo.urls[value] = key
	}

	trackerInfo.main = trackerInfo.urls[0]

	if len(trackerInfo.urls) == 0 {
		panic("No tcp/http tracker url announce found'")
	}
	return trackerInfo
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
