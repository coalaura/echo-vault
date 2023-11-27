@echo off

IF NOT EXIST bin mkdir bin

set GOOS=linux
set GOARCH=amd64

go build -o bin/echo_vault

set GOOS=windows
