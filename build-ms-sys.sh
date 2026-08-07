#!/bin/sh
set -eu

root="$(cd "$(dirname "$0")" && pwd)"
cd "$root/ms-sys"

make clean
# Build only the binary: the default target also builds .mo translation
# catalogs, which need msgfmt (gettext) and are never embedded anyway.
make CC="${CC:-gcc}" build/bin/ms-sys

mkdir -p "$root/binaries"
cp -a ./build/bin/ms-sys "$root/binaries/ms-sys"
