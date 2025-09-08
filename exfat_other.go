//go:build !linux

package main

import "errors"

func IsExFATAvailable() bool {
	return false
}

func MakeExFAT(device string) error {
	return errors.ErrUnsupported
}
