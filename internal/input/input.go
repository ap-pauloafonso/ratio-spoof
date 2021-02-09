package input

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/ap-pauloafonso/ratio-spoof/internal/bencode"
)

type InputArgs struct {
	TorrentPath       string
	InitialDownloaded string
	DownloadSpeed     string
	InitialUploaded   string
	UploadSpeed       string
	Port              int
	Debug             bool
}

type InputParsed struct {
	TorrentPath       string
	InitialDownloaded int
	DownloadSpeed     int
	InitialUploaded   int
	UploadSpeed       int
	Port              int
	Debug             bool
}

var validInitialSufixes = [...]string{"%", "b", "kb", "mb", "gb", "tb"}
var validSpeedSufixes = [...]string{"kbps", "mbps"}

func (I *InputArgs) ParseInput(torrentInfo *bencode.TorrentInfo) (*InputParsed, error) {
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

	return &InputParsed{InitialDownloaded: downloaded,
		DownloadSpeed:   downloadSpeed,
		InitialUploaded: uploaded,
		UploadSpeed:     uploadSpeed,
		Debug:           I.Debug,
		Port:            I.Port,
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
