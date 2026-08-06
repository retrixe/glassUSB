#!/bin/sh
set -euo pipefail

root="$(cd "$(dirname "$0")" && pwd)"
cd "$root/ms-sys"

make clean
make CC="${CC:-gcc}"

mkdir -p "$root/binaries"
cp -a ./build/bin/ms-sys "$root/binaries/ms-sys"
