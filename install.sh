#!/bin/bash

export GOPATH="`pwd`"

echo "creating workspace..."

# Setup working directory
echo "creating directories..."
if [ ! -d "`pwd`/src/tileserver" ]; then
    mkdir src/tileserver
fi
if [ ! -d "`pwd`/log" ]; then
    mkdir log
fi

# Move source files
echo "copying source files..."
cp -R tileserver/* src/

# sudo apt-get install libmapnik-dev
# cd mapnik/
# ./configure.bash
# cd ../
