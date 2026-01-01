@echo off

echo Building...

set GOOS=linux

go build -trimpath -buildvcs=false -ldflags "-s -w -X 'main.Version=exp'" -o echo_vault

set GOOS=windows

echo Done
