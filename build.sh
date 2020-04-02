#!/bin/bash
binName="bin/queueman"

GOOS=linux GOARCH=amd64 go build -o "$binName"_linux
GOOS=darwin GOARCH=amd64 go build -o "$binName"_macos
GOOS=freebsd GOARCH=amd64 go build -o "$binName"_freebsd_untest
GOOS=linux GOARCH=arm go build -o "$binName"_arm_untest
GOOS=windows GOARCH=amd64 go build -o "$binName"_win64_untest.exe

echo "build working is done, see below binary files"

ls "$binName"*
