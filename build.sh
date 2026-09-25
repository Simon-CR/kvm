#!/bin/bash
cd /root/picokvm-workspace
mkdir -p build
CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -tags=netgo -trimpath -ldflags="-X main.builtAppVersion=0.1.4-tesmart" -o build/kvm_app
