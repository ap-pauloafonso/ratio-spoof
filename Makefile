test:
	go test ./... -count=1 --cover

torrent-test:
	go run main.go -c qbit-4.3.3 -t bencode/torrent_files_test/debian-12.0.0-amd64-DVD-1.iso.torrent -d 0% -ds 100kbps -u 0% -us 100kbps

release:
	@if test -z "$(rsversion)"; then echo "usage: make release rsversion=v1.2"; exit 1; fi
	rm -rf ./out

	env GOOS=darwin GOARCH=amd64 go build -v -o ./out/mac/ratio-spoof .
	env GOOS=linux GOARCH=amd64 go build -v  -o ./out/linux/ratio-spoof .
	env GOOS=windows GOARCH=amd64 go build -v -o ./out/windows/ratio-spoof.exe .

	cd out/ ; zip ratio-spoof-$(rsversion)\(linux-mac-windows\).zip  -r .