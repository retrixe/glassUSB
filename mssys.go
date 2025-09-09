package main

import (
	_ "embed"
	"os"
)

//go:generate sh build-ms-sys.sh

//go:embed binaries/ms-sys
var MS_SYS_BIN []byte

func GetMsSysAsProgram() (*os.File, error) {
	f, err := os.CreateTemp(os.TempDir(), "ms-sys-*.bin")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if _, err := f.Write(MS_SYS_BIN); err != nil {
		return nil, err
	}

	if err := f.Chmod(0755); err != nil {
		return nil, err
	}

	return f, nil
}

func WriteMBR(location string) {
	// FIXME
}
