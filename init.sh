#!/bin/sh
set -e
apk add --update go git mercurial gcc build-base
mkdir -p /go/src/github.com/gliderlabs
cp -r /src /go/src/github.com/gliderlabs/logspout

cat > /src/modules.go <<END
package main

import (
    _ "github.com/mergermarket/logspout-logentries-autowire"
)
END

