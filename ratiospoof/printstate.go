package ratiospoof

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/olekukonko/ts"
)

func (R *ratioSpoofState) PrintState(exitedCH <-chan string) {
	exit := false
	go func() {
		_ = <-exitedCH
		exit = true
	}()

	for {
		if exit {
			break
		}
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
			var retryStr string
			if R.retryAttempt > 0 {
				retryStr = fmt.Sprintf("(*Retry %v - check your connection)", R.retryAttempt)
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
	Emulation: %v | Port: %v`, R.torrentInfo.Name, R.torrentInfo.TrackerInfo.Main, seedersStr, leechersStr, humanReadableSize(float64(R.input.downloadSpeed)),
				humanReadableSize(float64(R.input.uploadSpeed)), humanReadableSize(float64(R.torrentInfo.TotalSize)), R.bitTorrentClient.Name(), R.input.port)
			fmt.Printf("\n\n%s\n\n", center("  GITHUB.COM/AP-PAULOAFONSO/RATIO-SPOOF  ", width-len("  GITHUB.COM/AP-PAULOAFONSO/RATIO-SPOOF  "), "#"))
			for i := 0; i <= R.announceHistory.Len()-2; i++ {
				dequeItem := R.announceHistory.At(i).(announceEntry)
				fmt.Printf("#%v downloaded: %v(%.2f%%) | left: %v | uploaded: %v | announced\n", dequeItem.count, humanReadableSize(float64(dequeItem.downloaded)), dequeItem.percentDownloaded, humanReadableSize(float64(dequeItem.left)), humanReadableSize(float64(dequeItem.uploaded)))
			}
			lastDequeItem := R.announceHistory.At(R.announceHistory.Len() - 1).(announceEntry)
			fmt.Printf("#%v downloaded: %v(%.2f%%) | left: %v | uploaded: %v | next announce in: %v %v\n", lastDequeItem.count,
				humanReadableSize(float64(lastDequeItem.downloaded)),
				lastDequeItem.percentDownloaded,
				humanReadableSize(float64(lastDequeItem.left)),
				humanReadableSize(float64(lastDequeItem.uploaded)),
				fmtDuration(R.currentAnnounceTimer),
				retryStr)

			if R.input.debug {
				fmt.Printf("\n%s\n", center("  DEBUG  ", width-len("  DEBUG  "), "#"))
				fmt.Printf("\n%s\n\n%s", R.lastAnounceRequest, R.lastTackerResponse)
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

func fmtDuration(seconds int) string {
	d := time.Duration(seconds) * time.Second
	return fmt.Sprintf("%s", d)
}
