test:
	go test ./... --cover

torrent-test:
	go run cmd/main.go -c qbit-4.3.3 -t internal/bencode/torrent_files_test/Fedora-Workstation-Live-x86_64-33.torrent -d 0% -ds 100kbps -u 0% -us 100kbps -debug

release:
	@if test -z "$(rsversion)"; then echo "usage: make release rsversion=v1.2"; exit 1; fi
	rm -rf ./out

	env GOOS=darwin GOARCH=amd64 go build -v -o ./out/mac/ratio-spoof github.com/ap-pauloafonso/ratio-spoof/cmd
	env GOOS=linux GOARCH=amd64 go build -v  -o ./out/linux/ratio-spoof github.com/ap-pauloafonso/ratio-spoof/cmd
	env GOOS=windows GOARCH=amd64 go build -v -o ./out/windows/ratio-spoof.exe github.com/ap-pauloafonso/ratio-spoof/cmd

	cd out/ ; zip ratio-spoof-$(rsversion)\(linux-mac-windows\).zip  -r .