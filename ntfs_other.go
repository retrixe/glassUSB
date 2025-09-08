//go:build !linux

package main

import (
	"errors"
)

func IsNTFSAvailable() bool {
	return false
}

func MakeNTFS(device string) error {
	return errors.ErrUnsupported
}
