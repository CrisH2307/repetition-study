#!/bin/sh
# Usage: NAME="Product" ./build.sh   Writes dist/. Without NAME the page shows "Name pending".
set -eu
cd "$(dirname "$0")"
mkdir -p dist
sed "s/{{NAME}}/${NAME:-Name pending}/g" index.template.html > dist/index.html
cp style.css dist/
