package main

import (
	"flag"
	"fmt"
	"net/http"
	"runtime"

	"github.com/ap-pauloafonso/ratio-spoof/qbittorrent"
	"github.com/ap-pauloafonso/ratio-spoof/ratiospoof"
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
		var osExecutableSuffix string
		if runtime.GOOS == "windows" {
			fmt.Println(`usage: 
	ratio-spoof.exe -t <TORRENT_PATH> -d <INITIAL_DOWNLOADED> -ds <DOWNLOAD_SPEED> -u <INITIAL_UPLOADED> -us <UPLOAD_SPEED>`, osExecutableSuffix)
		} else {
			fmt.Println(`usage: 
	./ratio-spoof -t <TORRENT_PATH> -d <INITIAL_DOWNLOADED> -ds <DOWNLOAD_SPEED> -u <INITIAL_UPLOADED> -us <UPLOAD_SPEED>`, osExecutableSuffix)
		}

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

	qbit := qbittorrent.NewQbitTorrent()
	r, err := ratiospoof.NewRatioSPoofState(
		ratiospoof.InputArgs{
			TorrentPath:       *torrentPath,
			InitialDownloaded: *initialDownload,
			DownloadSpeed:     *downloadSpeed,
			InitialUploaded:   *initialUpload,
			UploadSpeed:       *uploadSpeed,
			Port:              *port,
			Debug:             *debug,
		},
		qbit,
		http.DefaultClient)

	if err != nil {
		panic(err)
	}
	r.Run()

}
