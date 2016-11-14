#!/bin/sh
set -x
mkdir -p /go/src/github.com/mergermarket/logspout-logentries-autowire
cp /logspout-logentries-autowire/logspout-logentries-autowire.go /go/src/github.com/mergermarket/logspout-logentries-autowire/logspout-logentries-autowire.go
cp /src/modules.go /go/src/github.com/gliderlabs/logspout/modules.go
cd /go/src/github.com/gliderlabs/logspout
export GOPATH=/go
go get -v
go build -ldflags "-X main.Version=1" -o /bin/logspout
if [ $? -eq 0 ]
then
    /bin/logspout  logentriesautowire://PUT-LOGENTRIES-TOKEN-FOR-TESTING-HERE
fi

