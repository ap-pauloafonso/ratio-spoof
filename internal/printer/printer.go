package printer

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/gammazero/deque"
	"github.com/olekukonko/ts"
)

type Printer struct {
	announceCount           *int
	seeders                 *int
	leechers                *int
	retryAttempt            *int
	downloadSpeed           *int
	uploadSpeed             *int
	port                    *int
	totalSize               *int
	emulation               *string
	torrentName             *string
	tracker                 *string
	lastAnnounceRequest     *string
	lastAnnounceResponse    *string
	debug                   *bool
	estimatedTimeToAnnounce *time.Time
	announceHistory         *deque.Deque
	print                   bool
}

func NewPrinter(announceCount, seeders, leechers, retryAttempt, downloadSpeed, uploadSpeed, port, totalSize *int,
	torrentName, tracker, emulation, lastAnnounceRequest, lastAnnounceResponse *string,
	debug *bool,
	estimatedTimeToAnnounce *time.Time,
	announceHistory *deque.Deque) *Printer {

	return &Printer{print: true,
		announceCount: announceCount, seeders: seeders, leechers: leechers, retryAttempt: retryAttempt, downloadSpeed: downloadSpeed, uploadSpeed: uploadSpeed, port: port, totalSize: totalSize,
		torrentName: torrentName, tracker: tracker, emulation: emulation, lastAnnounceRequest: lastAnnounceRequest,
		lastAnnounceResponse: lastAnnounceResponse, debug: debug, estimatedTimeToAnnounce: estimatedTimeToAnnounce, announceHistory: announceHistory,
	}
}

func (p *Printer) Start() {
	go func() {
		for {
			if !p.print {
				break
			}

			width := terminalSize()
			clear()

			if *p.announceCount == 1 {
				println("Trying to connect to the tracker...")
				time.Sleep(1 * time.Second)
				continue
			}
			if p.announceHistory.Len() > 0 {
				seedersStr := fmt.Sprint(*p.seeders)
				leechersStr := fmt.Sprint(*p.leechers)
				if *p.seeders == 0 {
					seedersStr = "not informed"
				}

				if *p.leechers == 0 {
					leechersStr = "not informed"
				}
				var retryStr string
				if *p.retryAttempt > 0 {
					retryStr = fmt.Sprintf("(*Retry %v - check your connection)", *p.retryAttempt)
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
		Emulation: %v | Port: %v`, *p.torrentName, *p.tracker, seedersStr, leechersStr, humanReadableSize(float64(*p.downloadSpeed)),
					humanReadableSize(float64(*p.uploadSpeed)), humanReadableSize(float64(*p.totalSize)), *p.emulation, *p.port)
				fmt.Printf("\n\n%s\n\n", center("  GITHUB.COM/AP-PAULOAFONSO/RATIO-SPOOF  ", width-len("  GITHUB.COM/AP-PAULOAFONSO/RATIO-SPOOF  "), "#"))
				for i := 0; i <= p.announceHistory.Len()-2; i++ {
					dequeItem := p.announceHistory.At(i).(struct {
						Count             int
						Downloaded        int
						PercentDownloaded float32
						Uploaded          int
						Left              int
					})
					fmt.Printf("#%v downloaded: %v(%.2f%%) | left: %v | uploaded: %v | announced\n", dequeItem.Count, humanReadableSize(float64(dequeItem.Downloaded)), dequeItem.PercentDownloaded, humanReadableSize(float64(dequeItem.Left)), humanReadableSize(float64(dequeItem.Uploaded)))
				}
				lastDequeItem := p.announceHistory.At(p.announceHistory.Len() - 1).(struct {
					Count             int
					Downloaded        int
					PercentDownloaded float32
					Uploaded          int
					Left              int
				})

				remaining := time.Until(*p.estimatedTimeToAnnounce)
				fmt.Printf("#%v downloaded: %v(%.2f%%) | left: %v | uploaded: %v | next announce in: %v %v\n", lastDequeItem.Count,
					humanReadableSize(float64(lastDequeItem.Downloaded)),
					lastDequeItem.PercentDownloaded,
					humanReadableSize(float64(lastDequeItem.Left)),
					humanReadableSize(float64(lastDequeItem.Uploaded)),
					fmtDuration(remaining),
					retryStr)

				if *p.debug {
					fmt.Printf("\n%s\n", center("  DEBUG  ", width-len("  DEBUG  "), "#"))
					fmt.Printf("\n%s\n\n%s", *p.lastAnnounceRequest, *p.lastAnnounceResponse)
				}
				time.Sleep(1 * time.Second)
			}
		}

	}()

}

func (p *Printer) Stop() {
	p.print = false
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
