GOBUILD=go build -x -v

default:
	go build ssh_exporter.go

linux:
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o 'ssh_exporter-linux-amd64' ssh_exporter.go

darwin:
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -o 'ssh_exporter-darwin-amd64' ssh_exporter.go

windows:
	GOOS=windows GOARCH=amd64 $(GOBUILD) -o 'ssh_exporter-windows-amd64' ssh_exporter.go

sha256sum:
	shasum -a 256 ssh_exporter-* > 'ssh_exporter-sha256.txt'

release: linux darwin windows sha256sum

.NOTPARALLEL:

.PHONY: default linux darwin windows sha256sum release
