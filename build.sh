#!/bin/sh

version=$1

if [ -z "$version" ]; then
  echo "usage: $0 <version>"
  exit 1
fi

rm -rf ./out

env GOOS=darwin GOARCH=amd64 go build -v -o ./out/mac/ratio-spoof github.com/ap-pauloafonso/ratio-spoof/cmd
env GOOS=linux GOARCH=amd64 go build -v  -o ./out/linux/ratio-spoof github.com/ap-pauloafonso/ratio-spoof/cmd
env GOOS=windows GOARCH=amd64 go build -v -o ./out/windows/ratio-spoof.exe github.com/ap-pauloafonso/ratio-spoof/cmd

cd out/
zip ratio-spoof-$version\(linux-mac-windows\).zip -r .