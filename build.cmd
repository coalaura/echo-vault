@echo off

IF NOT EXIST bin mkdir bin

set GOOS=linux
set GOARCH=amd64

go build -o bin/linux_amd64

set GOOS=windows

go build -o bin/win_amd64.exe