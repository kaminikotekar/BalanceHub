#!/bin/bash

PROJECT_DIR=$(dirname "$0")
BIN_DIR="$PROJECT_DIR/bin"
BIN_FILE="$BIN_DIR/bhub"

download_dependencies() {

    go mod download

}

build_binary() {

    download_dependencies
    local main_file="$PROJECT_DIR/cmd/main.go"
    if [ ! -d "$BIN_DIR" ]; then
        mkdir -p "$BIN_DIR"
        go build -tags sqlite_userauth -o "$1" "$main_file"
    elif [ ! -e "$1" ]; then
        go build -tags sqlite_userauth -o "$1" "$main_file"
    fi

}

run_project() {

    if [ ! -e "$1" ]; then
        build_binary "$1"
    fi
    ./"${1}"

}

force_build() {

    if [ -f "$1" ]; then
        rm "$1"
    fi
    build_binary "$1"

}

if [[ "$1" == "run" ]]; then
    run_project "$BIN_FILE"
elif [[ "$1" == "build" ]]; then
    build_binary "$BIN_FILE"
elif [[ "$1" == "force_build" ]]; then
    force_build "$BIN_FILE"
elif [[ "$1" == "--dockerbuild" ]]; then
    http_port=$(awk '/listen/ {print $2}' config.yaml)
    tcp_port=$(awk '/tcpListener/ {print $2}' config.yaml)
    echo "Running HTTP listener on $http_port and TCP listener on $tcp_port"
    docker build \
        --build-arg HTTP_PORT=${http_port} \
        --build-arg TCP_PORT=${tcp_port} \
        -t loadbalancer .
    docker run \
        -d \
        -p "$http_port:$http_port" \
        -p "$tcp_port:$tcp_port" \
        --name=lb loadbalancer
else
    echo "Invalid argument passed"
fi
