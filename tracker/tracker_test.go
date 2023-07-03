package tracker

import (
	"github.com/ap-pauloafonso/ratio-spoof/bencode"
	"reflect"
	"testing"
)

func TestNewHttpTracker(t *testing.T) {
	_, err := NewHttpTracker(&bencode.TorrentInfo{TrackerInfo: &bencode.TrackerInfo{Urls: []string{"udp://url1", "udp://url2"}}})
	got := err.Error()
	want := "No tcp/http tracker url announce found"

	if got != want {
		t.Errorf("got: %v want %v", got, want)
	}
}

func TestSwapFirst(t *testing.T) {
	tracker, _ := NewHttpTracker(&bencode.TorrentInfo{TrackerInfo: &bencode.TrackerInfo{Urls: []string{"http://url1", "http://url2", "http://url3", "http://url4"}}})
	tracker.swapFirst(3)

	got := tracker.Urls
	want := []string{"http://url4", "http://url2", "http://url3", "http://url1"}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got: %v want %v", got, want)
	}
}

func TestHandleSuccessfulResponse(t *testing.T) {

	t.Run("Empty interval should be overided with 1800 ", func(t *testing.T) {
		tracker, _ := NewHttpTracker(&bencode.TorrentInfo{TrackerInfo: &bencode.TrackerInfo{Urls: []string{"http://url1", "http://url2", "http://url3", "http://url4"}}})
		r := TrackerResponse{}
		tracker.handleSuccessfulResponse(&r)
		got := r.Interval
		want := 1800
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got: %v want %v", got, want)
		}

	})

	t.Run("Valid interval shouldn't be overwritten", func(t *testing.T) {
		tracker, _ := NewHttpTracker(&bencode.TorrentInfo{TrackerInfo: &bencode.TrackerInfo{Urls: []string{"http://url1", "http://url2", "http://url3", "http://url4"}}})
		r := TrackerResponse{Interval: 900}
		tracker.handleSuccessfulResponse(&r)
		got := r.Interval
		want := 900
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got: %v want %v", got, want)
		}

	})

}
