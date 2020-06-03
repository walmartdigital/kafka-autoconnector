#/bin/bash

set -e

$GOPATH/bin/golint -set_exit_status -min_confidence 0.81 ./...
$GOPATH/bin/gosec ./...
$GOPATH/bin/gocheckstyle -config=.go_style ./ server regex common
$GOPATH/bin/go-namecheck -rules .go_naming_rules ./...
$GOPATH/bin/dupl -t 200

go get -d -v ./...
go test -cover ./...