package bencode

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"regexp"
	"strconv"
)

const (
	dictToken                       = byte('d')
	numberToken                     = byte('i')
	listToken                       = byte('l')
	endOfCollectionToken            = byte('e')
	lengthValueStringSeparatorToken = byte(':')

	torrentInfoKey        = "info"
	torrentNameKey        = "name"
	torrentPieceLengthKey = "piece length"
	torrentLengthKey      = "length"
	torrentFilesKey       = "files"
	mainAnnounceKey       = "announce"
	announceListKey       = "announce-list"
	torrentDictOffsetsKey = "byte_offsets"
)

// TorrentInfo contains all relevant information extracted from a bencode file
type TorrentInfo struct {
	Name               string
	PieceSize          int
	TotalSize          int
	TrackerInfo        *TrackerInfo
	InfoHashURLEncoded string
}

//TrackerInfo contains http urls from the tracker
type TrackerInfo struct {
	Main string
	Urls []string
}

type torrentDict struct {
	resultMap map[string]interface{}
}

//TorrentDictParse decodes the bencoded bytes and builds the torrentInfo file
func TorrentDictParse(dat []byte) (torrent *TorrentInfo, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = e.(error)
		}
	}()

	dict, _ := mapParse(0, &dat)
	torrentMap := torrentDict{resultMap: dict}
	return &TorrentInfo{
		Name:               torrentMap.resultMap[torrentInfoKey].(map[string]interface{})[torrentNameKey].(string),
		PieceSize:          torrentMap.resultMap[torrentInfoKey].(map[string]interface{})[torrentPieceLengthKey].(int),
		TotalSize:          torrentMap.extractTotalSize(),
		TrackerInfo:        torrentMap.extractTrackerInfo(),
		InfoHashURLEncoded: torrentMap.extractInfoHashURLEncoded(dat),
	}, err
}

func (t *torrentDict) extractInfoHashURLEncoded(rawData []byte) string {
	byteOffsets := t.resultMap["info"].(map[string]interface{})["byte_offsets"].([]int)
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

func (t *torrentDict) extractTotalSize() int {
	if value, ok := t.resultMap[torrentInfoKey].(map[string]interface{})[torrentLengthKey]; ok {
		return value.(int)
	}
	var total int

	for _, file := range t.resultMap[torrentInfoKey].(map[string]interface{})[torrentFilesKey].([]interface{}) {
		total += file.(map[string]interface{})[torrentLengthKey].(int)
	}
	return total
}

func (t *torrentDict) extractTrackerInfo() *TrackerInfo {
	uniqueUrls := make(map[string]int)
	currentCount := 0
	if main, ok := t.resultMap[mainAnnounceKey]; ok {
		if _, found := uniqueUrls[main.(string)]; !found {
			uniqueUrls[main.(string)] = currentCount
			currentCount++
		}
	}
	if list, ok := t.resultMap[announceListKey]; ok {
		for _, innerList := range list.([]interface{}) {
			for _, item := range innerList.([]interface{}) {
				if _, found := uniqueUrls[item.(string)]; !found {
					uniqueUrls[item.(string)] = currentCount
					currentCount++
				}
			}
		}

	}
	trackerInfo := TrackerInfo{Urls: make([]string, len(uniqueUrls))}
	for key, value := range uniqueUrls {
		trackerInfo.Urls[value] = key
	}

	trackerInfo.Main = trackerInfo.Urls[0]
	return &trackerInfo
}

//Decode accepts a byte slice and returns a map with information parsed.
func Decode(data []byte) (dataMap map[string]interface{}, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = e.(error)
		}
	}()

	result, _ := findParse(0, &data)
	return result.(map[string]interface{}), err
}

func findParse(currentIdx int, data *[]byte) (result interface{}, nextIdx int) {
	token := (*data)[currentIdx : currentIdx+1][0]
	switch {
	case token == dictToken:
		return mapParse(currentIdx, data)
	case token == numberToken:
		return numberParse(currentIdx, data)
	case token == listToken:
		return listParse(currentIdx, data)
	case token >= byte('0') || token <= byte('9'):
		return stringParse(currentIdx, data)
	default:
		panic("Error decoding bencode")
	}
}

func mapParse(startIdx int, data *[]byte) (result map[string]interface{}, nextIdx int) {
	result = make(map[string]interface{})
	initialMapIndex := startIdx
	current := startIdx + 1
	for (*data)[current : current+1][0] != endOfCollectionToken {
		mapKey, next := findParse(current, data)
		current = next
		mapValue, next := findParse(current, data)
		current = next
		result[mapKey.(string)] = mapValue
	}
	current++
	result["byte_offsets"] = []int{initialMapIndex, current}
	nextIdx = current
	return
}

func listParse(startIdx int, data *[]byte) (result []interface{}, nextIdx int) {
	current := startIdx + 1
	for (*data)[current : current+1][0] != endOfCollectionToken {
		value, next := findParse(current, data)
		result = append(result, value)
		current = next
	}
	current++
	nextIdx = current
	return
}

func numberParse(startIdx int, data *[]byte) (result int, nextIdx int) {
	current := startIdx
	for (*data)[current : current+1][0] != endOfCollectionToken {
		current++
	}
	value, _ := strconv.Atoi(string((*data)[startIdx+1 : current]))
	result = value
	nextIdx = current + 1
	return
}

func stringParse(startIdx int, data *[]byte) (result string, nextIdx int) {
	current := startIdx
	for (*data)[current : current+1][0] != lengthValueStringSeparatorToken {
		current++
	}
	sizeStr, _ := strconv.Atoi(string(((*data)[startIdx:current])))
	result = string((*data)[current+1 : current+1+int(sizeStr)])
	nextIdx = current + 1 + int(sizeStr)
	return
}
