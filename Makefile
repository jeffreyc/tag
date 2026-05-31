SHELL:=/bin/bash

GOBUILD=go build -v

build:
	go build

generate_release_binaries:
	mkdir -p release; \
	GOOS=darwin GOARCH=386 ${GOBUILD} -o release/tag . && zip -j release/tag_darwin_386.zip release/tag; \
	GOOS=darwin GOARCH=amd64 ${GOBUILD} -o release/tag . && zip -j release/tag_darwin_amd64.zip release/tag; \
	GOOS=linux GOARCH=386 ${GOBUILD} -o release/tag . && tar -C release -cvzf release/tag_linux_386.tar.gz tag; \
	GOOS=linux GOARCH=amd64 ${GOBUILD} -o release/tag . && tar -C release -cvzf release/tag_linux_amd64.tar.gz tag; \
	GOOS=linux GOARCH=arm ${GOBUILD} -o release/tag . && tar -C release -cvzf release/tag_linux_arm.tar.gz tag; \
	GOOS=windows GOARCH=386 ${GOBUILD} -o release/tag.exe . && tar -C release -cvzf release/tag_windows_386.tar.gz tag.exe; \
	GOOS=windows GOARCH=amd64 ${GOBUILD} -o release/tag.exe . && tar -C release -cvzf release/tag_windows_amd64.tar.gz tag.exe;
