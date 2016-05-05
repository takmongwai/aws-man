#!/bin/bash

export GOOS=linux
export CGO_ENABLE=0
export GOARCH=amd64

go build -o ec2_snapshot ec2_snapshot.go
go build -o ec2_networkout_alerm ec2_networkout_alerm.go
