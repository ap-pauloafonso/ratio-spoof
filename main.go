package main

import (
	"flag"
	"fmt"
	"github.com/ap-pauloafonso/ratio-spoof/input"
	"github.com/ap-pauloafonso/ratio-spoof/printer"
	"github.com/ap-pauloafonso/ratio-spoof/ratiospoof"
	"log"
	"os"
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
	client := flag.String("c", "qbit-4.0.3", "emulated client")

	flag.Usage = func() {
		fmt.Printf("usage: %s -t <TORRENT_PATH> -d <INITIAL_DOWNLOADED> -ds <DOWNLOAD_SPEED> -u <INITIAL_UPLOADED> -us <UPLOAD_SPEED>\n", os.Args[0])
		fmt.Print(`
optional arguments:
	-h           		show this help message and exit
	-p [PORT]    		change the port number, default: 8999
	-c [CLIENT_CODE]	change the client emulation, default: qbit-4.0.3
	  
required arguments:
	-t  <TORRENT_PATH>     
	-d  <INITIAL_DOWNLOADED> 
	-ds <DOWNLOAD_SPEED>						  
	-u  <INITIAL_UPLOADED> 
	-us <UPLOAD_SPEED> 						  
	  
<INITIAL_DOWNLOADED> and <INITIAL_UPLOADED> must be in %, b, kb, mb, gb, tb
<DOWNLOAD_SPEED> and <UPLOAD_SPEED> must be in kbps, mbps
[CLIENT_CODE] options: qbit-4.0.3, qbit-4.3.3
`)
	}

	flag.Parse()

	if *torrentPath == "" || *initialDownload == "" || *downloadSpeed == "" || *initialUpload == "" || *uploadSpeed == "" {
		flag.Usage()
		return
	}

	r, err := ratiospoof.NewRatioSpoofState(
		input.InputArgs{
			TorrentPath:       *torrentPath,
			InitialDownloaded: *initialDownload,
			DownloadSpeed:     *downloadSpeed,
			InitialUploaded:   *initialUpload,
			UploadSpeed:       *uploadSpeed,
			Port:              *port,
			Debug:             *debug,
			Client:            *client,
		})

	if err != nil {
		log.Fatalln(err)
	}

	go printer.PrintState(r)
	r.Run()

}
