package ratiospoof

import (
	"compress/gzip"
	"crypto/sha1"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ap-pauloafonso/ratio-spoof/beencode"
	"github.com/gammazero/deque"
	"github.com/olekukonko/ts"
)

const (
	maxAnnounceHistory = 10
)

var validInitialSufixes = [...]string{"%", "b", "kb", "mb", "gb", "tb"}
var validSpeedSufixes = [...]string{"kbps"}

type ratioSPoofState struct {
	mutex                *sync.Mutex
	httpClient           HttpClient
	torrentInfo          *torrentInfo
	input                *inputParsed
	bitTorrentClient     TorrentClient
	announceRate         int
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
	downloadSpeed, err := extractInputKbpsSpeed(I.DownloadSpeed)
	if err != nil {
		return nil, err
	}
	uploadSpeed, err := extractInputKbpsSpeed(I.UploadSpeed)
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

func NewRatioSPoofState(input InputArgs, torrentClient TorrentClient, httpclient HttpClient) (*ratioSPoofState, error) {

	torrentInfo, err := extractTorrentInfo(input.TorrentPath)
	if err != nil {
		panic(err)
	}

	inputParsed, err := input.parseInput(torrentInfo)
	if err != nil {
		panic(err)
	}

	return &ratioSPoofState{
		bitTorrentClient: torrentClient,
		httpClient:       httpclient,
		torrentInfo:      torrentInfo,
		input:            inputParsed,
		numWant:          200,
		status:           "started",
		mutex:            &sync.Mutex{},
	}, nil
}

func checkSpeedSufix(input string) bool {
	for _, v := range validSpeedSufixes {

		if strings.HasSuffix(strings.ToLower(input), v) {
			return true
		}
	}
	return false
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
func extractInputKbpsSpeed(initialSpeedInput string) (int, error) {
	if !checkSpeedSufix(initialSpeedInput) {
		return 0, errors.New("speed must be in kbps")
	}
	number, _ := strconv.ParseFloat(initialSpeedInput[:len(initialSpeedInput)-4], 64)

	ret := int(number)

	if ret < 0 {
		return 0, errors.New("speed can not be negative")
	}
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

type TorrentClient interface {
	PeerID() string
	Key() string
	Query() string
	Name() string
	Headers() map[string]string
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

func (R *ratioSPoofState) Run() {
	R.firstAnnounce()
	go R.decreaseTimer()
	go R.printState()
	for {
		R.generateNextAnnounce()
		time.Sleep(time.Duration(R.announceInterval) * time.Second)
		R.fireAnnounce()
	}

}
func (R *ratioSPoofState) firstAnnounce() {
	println("Trying to connect to the tracker...")
	R.addAnnounce(R.input.initialDownloaded, R.input.initialUploaded, calculateBytesLeft(R.input.initialDownloaded, R.torrentInfo.totalSize), (float32(R.input.initialDownloaded)/float32(R.torrentInfo.totalSize))*100)
	R.fireAnnounce()
	R.AfterFirstRequestState()
}

func (R *ratioSPoofState) updateInterval(resp trackerResponse) {
	if resp.minInterval > 0 {
		R.announceInterval = resp.minInterval
	} else {
		R.announceInterval = resp.interval
	}
}

func (R *ratioSPoofState) updateSeedersAndLeechers(resp trackerResponse) {
	R.seeders = resp.seeders
	R.leechers = resp.leechers
}
func (R *ratioSPoofState) addAnnounce(currentDownloaded, currentUploaded, currentLeft int, percentDownloaded float32) {
	R.announceCount++
	R.announceHistory.pushValueHistory(announceEntry{count: R.announceCount, downloaded: currentDownloaded, uploaded: currentUploaded, left: currentLeft, percentDownloaded: percentDownloaded})
}
func (R *ratioSPoofState) fireAnnounce() {
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

	trackerResp := R.tryMakeRequest(query)
	R.updateSeedersAndLeechers(trackerResp)
	R.updateInterval(trackerResp)
}
func (R *ratioSPoofState) generateNextAnnounce() {
	R.resetTimer(R.announceInterval)
	lastAnnounce := R.announceHistory.Back().(announceEntry)
	currentDownloaded := lastAnnounce.downloaded
	var downloaded int
	if currentDownloaded < R.torrentInfo.totalSize {
		downloaded = calculateNextTotalSizeByte(R.input.downloadSpeed, currentDownloaded, R.torrentInfo.pieceSize, R.currentAnnounceTimer, R.torrentInfo.totalSize)
	} else {
		downloaded = R.torrentInfo.totalSize
	}

	currentUploaded := lastAnnounce.uploaded
	uploaded := calculateNextTotalSizeByte(R.input.uploadSpeed, currentUploaded, R.torrentInfo.pieceSize, R.currentAnnounceTimer, 0)

	left := calculateBytesLeft(downloaded, R.torrentInfo.totalSize)
	R.addAnnounce(downloaded, uploaded, left, (float32(downloaded)/float32(R.torrentInfo.totalSize))*100)
}
func (R *ratioSPoofState) decreaseTimer() {
	for {
		time.Sleep(1 * time.Second)
		R.mutex.Lock()
		if R.currentAnnounceTimer > 0 {
			R.currentAnnounceTimer--
		}
		R.mutex.Unlock()
	}
}
func (R *ratioSPoofState) printState() {
	terminalSize := func() int {
		size, _ := ts.GetSize()
		width := size.Col()
		if width < 40 {
			width = 40
		}
		return width
	}
	clear := func() {
		if runtime.GOOS == "windows" {
			cmd := exec.Command("cmd", "/c", "cls")
			cmd.Stdout = os.Stdout
			cmd.Run()
		} else {
			fmt.Print("\033c")
		}
	}

	center := func(s string, n int, fill string) string {
		div := n / 2
		return strings.Repeat(fill, div) + s + strings.Repeat(fill, div)
	}

	humanReadableSize := func(byteSize float64) string {
		var unitFound string
		for _, unit := range []string{"B", "KiB", "MiB", "GiB", "TiB"} {
			if byteSize < 1024.0 {
				unitFound = unit
				break
			}
			byteSize /= 1024.0
		}
		return fmt.Sprintf("%.2f%v", byteSize, unitFound)
	}

	fmtDuration := func(seconds int) string {
		d := time.Duration(seconds) * time.Second
		return fmt.Sprintf("%s", d)
	}

	for {
		width := terminalSize()
		clear()
		if R.announceHistory.Len() > 0 {
			seedersStr := fmt.Sprint(R.seeders)
			leechersStr := fmt.Sprint(R.leechers)
			if R.seeders == 0 {
				seedersStr = "not informed"
			}

			if R.leechers == 0 {
				leechersStr = "not informed"
			}
			fmt.Println(center("  RATIO-SPOOF  ", width-len("  RATIO-SPOOF  "), "#"))
			fmt.Printf(`
	Torrent: %v
	Tracker: %v
	Seeders: %v
	Leechers:%v
	Download Speed: %vKB/s
	Upload Speed: %vKB/s
	Size: %v
	Emulation: %v | Port: %v`, R.torrentInfo.name, R.torrentInfo.trackerInfo.main, seedersStr, leechersStr, R.input.downloadSpeed,
				R.input.uploadSpeed, humanReadableSize(float64(R.torrentInfo.totalSize)), R.bitTorrentClient.Name(), R.input.port)
			fmt.Println()
			fmt.Println()
			fmt.Println(center("  GITHUB.COM/AP-PAULOAFONSO/RATIO-SPOOF  ", width-len("  GITHUB.COM/AP-PAULOAFONSO/RATIO-SPOOF  "), "#"))
			fmt.Println()
			for i := 0; i <= R.announceHistory.Len()-2; i++ {
				dequeItem := R.announceHistory.At(i).(announceEntry)
				fmt.Printf("#%v downloaded: %v(%.2f%%) | left: %v | uploaded: %v | announced", dequeItem.count, humanReadableSize(float64(dequeItem.downloaded)), dequeItem.percentDownloaded, humanReadableSize(float64(dequeItem.left)), humanReadableSize(float64(dequeItem.uploaded)))
				fmt.Println()

			}
			lastDequeItem := R.announceHistory.At(R.announceHistory.Len() - 1).(announceEntry)
			fmt.Printf("#%v downloaded: %v(%.2f%%) | left: %v | uploaded: %v | next announce in: %v", lastDequeItem.count, humanReadableSize(float64(lastDequeItem.downloaded)), lastDequeItem.percentDownloaded, humanReadableSize(float64(lastDequeItem.left)), humanReadableSize(float64(lastDequeItem.uploaded)), fmtDuration(R.currentAnnounceTimer))

			if R.input.debug {
				fmt.Println()
				fmt.Println()
				fmt.Println(center("  DEBUG  ", width-len("  DEBUG  "), "#"))
				fmt.Println()
				fmt.Print(R.lastAnounceRequest)
				fmt.Println()
				fmt.Println()
				fmt.Print(R.lastTackerResponse)
			}
			time.Sleep(1 * time.Second)
		}
	}
}

func (R *ratioSPoofState) resetTimer(newAnnounceRate int) {
	R.announceRate = newAnnounceRate
	R.mutex.Lock()
	defer R.mutex.Unlock()
	R.currentAnnounceTimer = R.announceRate
}

func (R *ratioSPoofState) tryMakeRequest(query string) trackerResponse {
	for idx, url := range R.torrentInfo.trackerInfo.urls {
		completeURL := url + "?" + strings.TrimLeft(query, "?")
		R.lastAnounceRequest = completeURL
		req, _ := http.NewRequest("GET", completeURL, nil)
		for header, value := range R.bitTorrentClient.Headers() {
			req.Header.Add(header, value)
		}
		resp, err := R.httpClient.Do(req)
		if err == nil {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				bytes, _ := ioutil.ReadAll(resp.Body)
				mimeType := http.DetectContentType(bytes)
				if mimeType == "application/x-gzip" {
					gzipReader, _ := gzip.NewReader(resp.Body)
					defer gzipReader.Close()
					bytes, _ = ioutil.ReadAll(gzipReader)
				}
				R.lastTackerResponse = string(bytes)
				decodedResp := beencode.Decode(bytes)
				if idx != 0 {
					R.torrentInfo.trackerInfo.SwapFirst(idx)
				}
				ret := extractTrackerResponse(decodedResp)
				return ret
			}
		}
	}
	panic("Connection error with the tracker")

}
func (R *ratioSPoofState) AfterFirstRequestState() {
	R.status = ""
}

func calculateNextTotalSizeByte(speedKbps, currentByte, pieceSizeByte, seconds, limitTotalBytes int) int {
	if speedKbps == 0 {
		return currentByte
	}
	total := currentByte + (speedKbps * 1024 * seconds)
	closestPieceNumber := int(total / pieceSizeByte)
	closestPieceNumber += 5
	nextTotal := closestPieceNumber * pieceSizeByte

	if limitTotalBytes != 0 && nextTotal > limitTotalBytes {
		return limitTotalBytes
	}
	return nextTotal
}

func extractInfoHashURLEncoded(rawData []byte, torrentData map[string]interface{}) string {
	byteOffsets := torrentData["info"].(map[string]interface{})["byte_offsets"].([]int)
	h := sha1.New()
	h.Write([]byte(rawData[byteOffsets[0]:byteOffsets[1]]))
	ret := h.Sum(nil)
	return url.QueryEscape(string(ret))

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
