#!/bin/bash

export GOPATH="`pwd`"

echo "creating workspace..."

# Setup working directory
echo "creating directories..."
if [ ! -d "`pwd`/bin" ]; then
    mkdir bin
fi
if [ ! -d "`pwd`/pkg" ]; then
    mkdir pkg
fi
if [ ! -d "`pwd`/src" ]; then
    mkdir src
fi
if [ ! -d "`pwd`/src/tileserver" ]; then
    mkdir src/tileserver
fi
if [ ! -d "`pwd`/log" ]; then
    mkdir log
fi

# Move source files
echo "copying source files..."
cp -R tileserver/* src/tileserver/

# Download required libraries
echo "checking requirements..."
if [ ! -d "`pwd`/src/github.com/mattn/go-sqlite3" ]; then
    echo "downloading go-sqlite3..."
    go get github.com/mattn/go-sqlite3
fi

if [ ! -d "`pwd`/src/github.com/lib/pq" ]; then
    echo "downloading pg..."
    go get github.com/lib/pq
fi

if [ ! -d "`pwd`/src/github.com/cihub/seelog" ]; then
    echo "downloading seelog..."
    go get github.com/cihub/seelog
fi

if [ ! -d "`pwd`/src/github.com/gorilla/mux" ]; then
    echo "downloading seelog..."
    go get github.com/gorilla/mux
fi


# sudo apt-get install libmapnik-dev
# cd mapnik/
# ./configure.bash
# cd ../
