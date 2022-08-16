#!/bin/sh

set -ex

mkdir -p .tmp
go build -ldflags "-extldflags \"-s -w -static\"" -o .tmp/awecloud-btel-sdk.out .