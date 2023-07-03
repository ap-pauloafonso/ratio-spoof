package input

import (
	"errors"
	"fmt"
	"github.com/ap-pauloafonso/ratio-spoof/bencode"
	"math"
	"strconv"
	"strings"
)

const (
	minPortNumber     = 1
	maxPortNumber     = 65535
	speedSuffixLength = 4
)

type InputArgs struct {
	TorrentPath       string
	InitialDownloaded string
	DownloadSpeed     string
	InitialUploaded   string
	Client            string
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

func (i *InputArgs) ParseInput(torrentInfo *bencode.TorrentInfo) (*InputParsed, error) {
	downloaded, err := extractInputInitialByteCount(i.InitialDownloaded, torrentInfo.TotalSize, true)
	if err != nil {
		return nil, err
	}
	uploaded, err := extractInputInitialByteCount(i.InitialUploaded, torrentInfo.TotalSize, false)
	if err != nil {
		return nil, err
	}
	downloadSpeed, err := extractInputByteSpeed(i.DownloadSpeed)
	if err != nil {
		return nil, err
	}
	uploadSpeed, err := extractInputByteSpeed(i.UploadSpeed)
	if err != nil {
		return nil, err
	}

	if i.Port < minPortNumber || i.Port > maxPortNumber {
		return nil, errors.New(fmt.Sprint("port number must be between %i and %i", minPortNumber, maxPortNumber))
	}

	return &InputParsed{InitialDownloaded: downloaded,
		DownloadSpeed:   downloadSpeed,
		InitialUploaded: uploaded,
		UploadSpeed:     uploadSpeed,
		Debug:           i.Debug,
		Port:            i.Port,
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
	byteCount, err := strSize2ByteSize(initialSizeInput, totalBytes)
	if err != nil {
		return 0, err
	}
	if errorIfHigher && byteCount > totalBytes {
		return 0, errors.New("initial downloaded can not be higher than the torrent size")
	}
	if byteCount < 0 {
		return 0, errors.New("initial value can not be negative")
	}
	return byteCount, nil
}

// Takes an dirty speed input and returns the bytes per second based on the suffixes
// example 1kbps(string) > 1024 bytes per second (int)
func extractInputByteSpeed(initialSpeedInput string) (int, error) {
	ok, suffix := checkSpeedSufix(initialSpeedInput)
	if !ok {
		return 0, fmt.Errorf("speed must be in %v", validSpeedSufixes)
	}
	speedVal, err := strconv.ParseFloat(initialSpeedInput[:len(initialSpeedInput)-speedSuffixLength], 64)
	if err != nil {
		return 0, errors.New("invalid speed number")
	}
	if speedVal < 0 {
		return 0, errors.New("speed can not be negative")
	}

	if suffix == "kbps" {
		speedVal *= 1024
	} else {
		speedVal = speedVal * 1024 * 1024
	}
	ret := int(speedVal)
	return ret, nil
}

func extractByteSizeNumber(strWithSufix string, sufixLength, power int) (int, error) {
	v, err := strconv.ParseFloat(strWithSufix[:len(strWithSufix)-sufixLength], 64)
	if err != nil {
		return 0, err
	}
	result := v * math.Pow(1024, float64(power))
	return int(result), nil
}

func strSize2ByteSize(input string, totalSize int) (int, error) {
	lowerInput := strings.ToLower(input)
	invalidSizeError := errors.New("invalid input size")
	switch {
	case strings.HasSuffix(lowerInput, "kb"):
		{
			v, err := extractByteSizeNumber(lowerInput, 2, 1)
			if err != nil {
				return 0, invalidSizeError
			}
			return v, nil
		}
	case strings.HasSuffix(lowerInput, "mb"):
		{
			v, err := extractByteSizeNumber(lowerInput, 2, 2)
			if err != nil {
				return 0, invalidSizeError
			}
			return v, nil
		}
	case strings.HasSuffix(lowerInput, "gb"):
		{
			v, err := extractByteSizeNumber(lowerInput, 2, 3)
			if err != nil {
				return 0, invalidSizeError
			}
			return v, nil
		}
	case strings.HasSuffix(lowerInput, "tb"):
		{
			v, err := extractByteSizeNumber(lowerInput, 2, 4)
			if err != nil {
				return 0, invalidSizeError
			}
			return v, nil
		}
	case strings.HasSuffix(lowerInput, "b"):
		{
			v, err := extractByteSizeNumber(lowerInput, 1, 0)
			if err != nil {
				return 0, invalidSizeError
			}
			return v, nil
		}
	case strings.HasSuffix(lowerInput, "%"):
		{
			v, err := strconv.ParseFloat(lowerInput[:len(lowerInput)-1], 64)
			if v < 0 || v > 100 || err != nil {
				return 0, errors.New("percent value must be in (0-100)")
			}
			result := int(float64(v/100) * float64(totalSize))

			return result, nil
		}

	default:
		return 0, errors.New("Size not found")
	}
}
