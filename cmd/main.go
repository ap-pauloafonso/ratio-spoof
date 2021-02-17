package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/ap-pauloafonso/ratio-spoof/internal/emulation"
	"github.com/ap-pauloafonso/ratio-spoof/internal/input"
	"github.com/ap-pauloafonso/ratio-spoof/internal/printer"
	"github.com/ap-pauloafonso/ratio-spoof/internal/ratiospoof"
)

func main() {

	//required
	torrentPath := flag.String("t", "", "torrent path")
	initialDownload := flag.String("d", "", "a INITIAL_DOWNLOADED")
	downloadSpeed := flag.String("ds", "", "a DOWNLOAD_SPEED")
	initialUpload := flag.String("u", "", "a INITIAL_UPLOADED")
	uploadSpeed := flag.String("us", "", "a UPLOAD_SPEED")

	//optional
	port := flag.Int("p", 8999, "a PORT")
	debug := flag.Bool("debug", false, "")

	flag.Usage = func() {
		fmt.Printf("usage: %s -t <TORRENT_PATH> -d <INITIAL_DOWNLOADED> -ds <DOWNLOAD_SPEED> -u <INITIAL_UPLOADED> -us <UPLOAD_SPEED>\n", os.Args[0])
		fmt.Print(`
optional arguments:
	-h           		show this help message and exit
	-p [PORT]    		change the port number, the default is 8999
	  
required arguments:
	-t  <TORRENT_PATH>     
	-d  <INITIAL_DOWNLOADED> 
	-ds <DOWNLOAD_SPEED>						  
	-u  <INITIAL_UPLOADED> 
	-us <UPLOAD_SPEED> 						  
	  
<INITIAL_DOWNLOADED> and <INITIAL_UPLOADED> must be in %, b, kb, mb, gb, tb
<DOWNLOAD_SPEED> and <UPLOAD_SPEED> must be in kbps, mbps
`)
	}

	flag.Parse()

	if *torrentPath == "" || *initialDownload == "" || *downloadSpeed == "" || *initialUpload == "" || *uploadSpeed == "" {
		flag.Usage()
		return
	}

	qbit := emulation.NewQbitTorrent()
	r, err := ratiospoof.NewRatioSpoofState(
		input.InputArgs{
			TorrentPath:       *torrentPath,
			InitialDownloaded: *initialDownload,
			DownloadSpeed:     *downloadSpeed,
			InitialUploaded:   *initialUpload,
			UploadSpeed:       *uploadSpeed,
			Port:              *port,
			Debug:             *debug,
		},
		qbit)

	if err != nil {
		log.Fatalln(err)
	}

	go printer.PrintState(r)
	r.Run()

}
