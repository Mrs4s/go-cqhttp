#!/usr/bin/env bash
#
# automatically generated in linux environment
#
# automatically generate version information:
#   go build -ldflags "-X path.varname=varvalue" -o filename .
#
# shell use e.g:
#   ./build.sh v1.0.0
#   /get_version_info => version:v1.0.0
#
# all comb to compile
# $GOOS        $GOARCH
# android     arm
# darwin      386
# darwin      amd64
# darwin      arm
# darwin      arm64
# dragonfly   amd64
# freebsd     386
# freebsd     amd64
# freebsd     arm
# linux       386
# linux       amd64
# linux       arm
# linux       arm64
# linux       ppc64
# linux       ppc64le
# linux       mips
# linux       mipsle
# linux       mips64
# linux       mips64le
# netbsd      386
# netbsd      amd64
# netbsd      arm
# openbsd     386
# openbsd     amd64
# openbsd     arm
# plan9       386
# plan9       amd64
# solaris     amd64
# windows     386
# windows     amd64

function build_linux32() {
    export GOOS=linux
    export GOARCH=386
    filename=go-cqhttp-"$1"-linux-386
    go build -ldflags "-X github.com/Mrs4s/go-cqhttp/coolq.version=$1" -o "$filename" .
    tar zcvf "$filename".tar.gz "$filename" --remove-files
    md5sum "$filename".tar.gz > "$filename".tar.gz.md5
    mv "$filename".tar.gz ./dist
    mv "$filename".tar.gz.md5 ./dist
}

function build_linux64() {
    export GOOS=linux
    export GOARCH=amd64
    filename=go-cqhttp-"$1"-linux-amd64
    go build -ldflags "-X github.com/Mrs4s/go-cqhttp/coolq.version=$1" -o "$filename" .
    tar zcvf "$filename".tar.gz "$filename" --remove-files
    md5sum "$filename".tar.gz > "$filename".tar.gz.md5
    mv "$filename".tar.gz ./dist
    mv "$filename".tar.gz.md5 ./dist
}

function build_win32() {
    export GOOS=windows
    export GOARCH=386
    filename=go-cqhttp-"$1"-windows-386
    go build -ldflags "-X github.com/Mrs4s/go-cqhttp/coolq.version=$1" -o "$filename" .
    tar zcvf "$filename".tar.gz "$filename" --remove-files
    md5sum "$filename".tar.gz > "$filename".tar.gz.md5
    mv "$filename".tar.gz ./dist
    mv "$filename".tar.gz.md5 ./dist
}

function build_win64() {
    export GOOS=windows
    export GOARCH=amd64
    filename=go-cqhttp-"$1"-windows-amd64
    go build -ldflags "-X github.com/Mrs4s/go-cqhttp/coolq.version=$1" -o "$filename" .
    tar zcvf "$filename".tar.gz "$filename"
    md5sum "$filename".tar.gz > "$filename".tar.gz.md5
    mv "$filename".tar.gz ./dist
    mv "$filename".tar.gz.md5 ./dist
}

function build_darwin32() {
    export GOOS=darwin
    export GOARCH=386
    filename=go-cqhttp-"$1"-darwin-386
    go build -ldflags "-X github.com/Mrs4s/go-cqhttp/coolq.version=$1" -o go-cqhttp-"$1"-darwin-386 .
    tar zcvf "$filename".tar.gz "$filename" --remove-files
    md5sum "$filename".tar.gz > "$filename".tar.gz.md5
    mv "$filename".tar.gz ./dist
    mv "$filename".tar.gz.md5 ./dist
}

function build_darwin64() {
    export GOOS=darwin
    export GOARCH=amd64
    filename=go-cqhttp-"$1"-darwin-amd64
    go build -ldflags "-X github.com/Mrs4s/go-cqhttp/coolq.version=$1" -o go-cqhttp-"$1"-darwin-amd64 .
    tar zcvf "$filename".tar.gz "$filename" --remove-files
    md5sum "$filename".tar.gz > "$filename".tar.gz.md5
    mv "$filename".tar.gz ./dist
    mv "$filename".tar.gz.md5 ./dist
}

function main() {
    if [ ! -d 'dist'  ];then
        mkdir dist
    fi

    #build_linux32 $1
    build_linux64 $1
    #build_win32 $1
    #build_win64 $1
    #build_darwin32 $1
    #build_darwin64 $1

}


if [ -n "$1" ]; then
    main $1
else
    echo "No version info input...exit!"
fi