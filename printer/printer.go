package printer

import (
	"fmt"
	"github.com/ap-pauloafonso/ratio-spoof/ratiospoof"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/olekukonko/ts"
)

func PrintState(state *ratiospoof.RatioSpoof) {
	for {
		if !state.Print {
			break
		}
		width := terminalSize()
		clear()

		if state.AnnounceCount == 1 {
			println("Trying to connect to the tracker...")
			time.Sleep(1 * time.Second)
			continue
		}
		if state.AnnounceHistory.Len() > 0 {
			seedersStr := fmt.Sprint(state.Seeders)
			leechersStr := fmt.Sprint(state.Leechers)
			if state.Seeders == 0 {
				seedersStr = "not informed"
			}

			if state.Leechers == 0 {
				leechersStr = "not informed"
			}
			var retryStr string
			if state.Tracker.RetryAttempt > 0 {
				retryStr = fmt.Sprintf("(*Retry %v - check your connection)", state.Tracker.RetryAttempt)
			}
			fmt.Printf("%s\n", center("  RATIO-SPOOF  ", width-len("  RATIO-SPOOF  "), "#"))
			fmt.Printf(`
	Torrent: %v
	Tracker: %v
	Seeders: %v
	Leechers:%v
	Download Speed: %v/s
	Upload Speed: %v/s
	Size: %v
	Emulation: %v | Port: %v`, state.TorrentInfo.Name, state.TorrentInfo.TrackerInfo.Main, seedersStr, leechersStr, humanReadableSize(float64(state.Input.DownloadSpeed)),
				humanReadableSize(float64(state.Input.UploadSpeed)), humanReadableSize(float64(state.TorrentInfo.TotalSize)), state.BitTorrentClient.Name, state.Input.Port)
			fmt.Printf("\n\n%s\n\n", center("  GITHUB.COM/AP-PAULOAFONSO/RATIO-SPOOF  ", width-len("  GITHUB.COM/AP-PAULOAFONSO/RATIO-SPOOF  "), "#"))
			for i := 0; i <= state.AnnounceHistory.Len()-2; i++ {
				dequeItem := state.AnnounceHistory.At(i).(ratiospoof.AnnounceEntry)
				fmt.Printf("#%v downloaded: %v(%.2f%%) | left: %v | uploaded: %v | announced\n", dequeItem.Count, humanReadableSize(float64(dequeItem.Downloaded)), dequeItem.PercentDownloaded, humanReadableSize(float64(dequeItem.Left)), humanReadableSize(float64(dequeItem.Uploaded)))
			}
			lastDequeItem := state.AnnounceHistory.At(state.AnnounceHistory.Len() - 1).(ratiospoof.AnnounceEntry)

			remaining := time.Until(state.Tracker.EstimatedTimeToAnnounce)
			fmt.Printf("#%v downloaded: %v(%.2f%%) | left: %v | uploaded: %v | next announce in: %v %v\n", lastDequeItem.Count,
				humanReadableSize(float64(lastDequeItem.Downloaded)),
				lastDequeItem.PercentDownloaded,
				humanReadableSize(float64(lastDequeItem.Left)),
				humanReadableSize(float64(lastDequeItem.Uploaded)),
				fmtDuration(remaining),
				retryStr)

			if state.Input.Debug {
				fmt.Printf("\n%s\n", center("  DEBUG  ", width-len("  DEBUG  "), "#"))
				fmt.Printf("\n%s\n\n%s", state.Tracker.LastAnounceRequest, state.Tracker.LastTackerResponse)
			}
			time.Sleep(1 * time.Second)
		}
	}
}

func terminalSize() int {
	size, _ := ts.GetSize()
	width := size.Col()
	if width < 40 {
		width = 40
	}
	return width
}
func clear() {
	if runtime.GOOS == "windows" {
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
	} else {
		fmt.Print("\033c")
	}
}

func center(s string, n int, fill string) string {
	div := n / 2
	return strings.Repeat(fill, div) + s + strings.Repeat(fill, div)
}

func humanReadableSize(byteSize float64) string {
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

func fmtDuration(d time.Duration) string {
	if d.Seconds() < 0 {
		return fmt.Sprintf("%s", 0*time.Second)
	}
	return fmt.Sprintf("%s", time.Duration(int(d.Seconds()))*time.Second)
}
