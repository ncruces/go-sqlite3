#!/usr/bin/env bash
set -euo pipefail

echo android       ; GOOS=android   GOARCH=amd64 go build .
echo darwin        ; GOOS=darwin    GOARCH=amd64 go build .
echo dragonfly     ; GOOS=dragonfly GOARCH=amd64 go build .
echo freebsd       ; GOOS=freebsd   GOARCH=amd64 go build .
echo illumos       ; GOOS=illumos   GOARCH=amd64 go build .
echo ios           ; GOOS=ios       GOARCH=amd64 go build .
echo linux         ; GOOS=linux     GOARCH=amd64 go build .
echo netbsd        ; GOOS=netbsd    GOARCH=amd64 go build .
echo openbsd       ; GOOS=openbsd   GOARCH=amd64 go build .
echo plan9         ; GOOS=plan9     GOARCH=amd64 go build .
echo solaris       ; GOOS=solaris   GOARCH=amd64 go build .
echo windows       ; GOOS=windows   GOARCH=amd64 go build .
echo aix           ; GOOS=aix       GOARCH=ppc64 go build .
echo js            ; GOOS=js        GOARCH=wasm  go build .
echo wasip1        ; GOOS=wasip1    GOARCH=wasm  go build .
echo linux-flock   ; GOOS=linux     GOARCH=amd64 go build -tags sqlite3_flock .
echo linux-noshm   ; GOOS=linux     GOARCH=amd64 go build -tags sqlite3_noshm .
echo linux-nosys   ; GOOS=linux     GOARCH=amd64 go build -tags sqlite3_nosys .
echo darwin-flock  ; GOOS=darwin    GOARCH=amd64 go build -tags sqlite3_flock .
echo darwin-noshm  ; GOOS=darwin    GOARCH=amd64 go build -tags sqlite3_noshm .
echo darwin-nosys  ; GOOS=darwin    GOARCH=amd64 go build -tags sqlite3_nosys .
echo windows-nosys ; GOOS=windows   GOARCH=amd64 go build -tags sqlite3_nosys .
echo freebsd-nosys ; GOOS=freebsd   GOARCH=amd64 go build -tags sqlite3_nosys .
echo solaris-flock ; GOOS=solaris   GOARCH=amd64 go build -tags sqlite3_flock .