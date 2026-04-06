#!/bin/sh
set -e

go build -o gitwit .
sudo cp gitwit /usr/local/bin/gitwit
echo "gitwit installed to /usr/local/bin/gitwit"
