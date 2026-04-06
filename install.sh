#!/bin/sh
set -e

go build -o gitree .
sudo cp gitree /usr/local/bin/gitree
echo "gitree installed to /usr/local/bin/gitree"
