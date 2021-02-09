package tracker

import (
	"bytes"
	"compress/gzip"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/ap-pauloafonso/ratio-spoof/internal/bencode"
)

type HttpTracker struct {
	Urls               []string
	RetryAttempt       int
	LastAnounceRequest string
	LastTackerResponse string
}

type TrackerResponse struct {
	MinInterval int
	Interval    int
	Seeders     int
	Leechers    int
}

func NewHttpTracker(torrentInfo *bencode.TorrentInfo, timerChangeChannel chan<- int) (*HttpTracker, error) {

	var result []string
	for _, url := range torrentInfo.TrackerInfo.Urls {
		if strings.HasPrefix(url, "http") {
			result = append(result, url)
		}
	}
	if len(result) == 0 {
		return nil, errors.New("No tcp/http tracker url announce found")
	}
	return &HttpTracker{Urls: torrentInfo.TrackerInfo.Urls}, nil
}

func (T *HttpTracker) SwapFirst(currentIdx int) {
	aux := T.Urls[0]
	T.Urls[0] = T.Urls[currentIdx]
	T.Urls[currentIdx] = aux
}

func (T *HttpTracker) Announce(query string, headers map[string]string, retry bool, timerUpdateChannel chan<- int) *TrackerResponse {
	var trackerResp *TrackerResponse
	if retry {
		retryDelay := 30 * time.Second
		for {
			exit := false
			func() {
				defer func() {
					if err := recover(); err != nil {
						timerUpdateChannel <- int(retryDelay.Seconds())
						T.RetryAttempt++
						time.Sleep(retryDelay)
						retryDelay *= 2
						if retryDelay.Seconds() > 900 {
							retryDelay = 900
						}
					}
				}()
				trackerResp = T.tryMakeRequest(query, headers)
				exit = true
			}()
			if exit {
				break
			}
		}

	} else {
		trackerResp = T.tryMakeRequest(query, headers)
	}
	T.RetryAttempt = 0

	return trackerResp

}

func (t *HttpTracker) tryMakeRequest(query string, headers map[string]string) *TrackerResponse {
	for idx, baseUrl := range t.Urls {
		completeURL := buildFullUrl(baseUrl, query)
		t.LastAnounceRequest = completeURL
		req, _ := http.NewRequest("GET", completeURL, nil)
		for header, value := range headers {
			req.Header.Add(header, value)
		}
		resp, err := http.DefaultClient.Do(req)
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
				t.LastTackerResponse = string(bytesR)
				decodedResp := bencode.Decode(bytesR)
				if idx != 0 {
					t.SwapFirst(idx)
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

func extractTrackerResponse(datatrackerResponse map[string]interface{}) TrackerResponse {
	var result TrackerResponse
	if v, ok := datatrackerResponse["failure reason"].(string); ok && len(v) > 0 {
		panic(errors.New(v))
	}
	result.MinInterval, _ = datatrackerResponse["min interval"].(int)
	result.Interval, _ = datatrackerResponse["interval"].(int)
	result.Seeders, _ = datatrackerResponse["complete"].(int)
	result.Leechers, _ = datatrackerResponse["incomplete"].(int)
	return result

}
