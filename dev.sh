#!/bin/sh

set -e

docker run \
    -v $PWD:/logspout-logentries-autowire \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -i -t --entrypoint=/bin/sh \
    --rm \
    --label logentries.access-key='my_access_key' \
    --name logspout-logentries-autowire \
    gliderlabs/logspout:master


