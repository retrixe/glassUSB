//go:build !linux

package main

import "errors"

func IsFAT32Available() bool {
	return false
}

func MakeFAT32(device string) error {
	return errors.ErrUnsupported
}

func IsExFATAvailable() bool {
	return false
}

func MakeExFAT(device string) error {
	return errors.ErrUnsupported
}

func IsNTFSAvailable() bool {
	return false
}

func MakeNTFS(device string) error {
	return errors.ErrUnsupported
}
