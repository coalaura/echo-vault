@echo off

if NOT EXIST bin (
    mkdir bin
)

echo Building...
set GOOS=linux
go build -o bin/echo_vault
set GOOS=windows

echo Done.