#!/bin/bash
set -euo pipefail
cd "$(dirname "$0")"
mkdir -p build
APP_VERSION="${APP_VERSION:-0.1.5+tesmart}"
OTA_PUBLIC_KEY="${OTA_PUBLIC_KEY:-4d78341c5c66fd5c09635d45ad3aa0ae7ab131f9e945868ec9726fbc1367a452}"
CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -tags=netgo -trimpath \
  -ldflags="-s -w -X kvm.builtAppVersion=${APP_VERSION} -X kvm.builtOtaPublicKey=${OTA_PUBLIC_KEY}" \
  -o build/kvm_app ./cmd/main.go
