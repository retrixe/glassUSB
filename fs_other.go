//go:build !linux && !darwin

package main

import "errors"

func IsFAT32Available() bool {
	return false
}

func MakeFAT32(device string, label string) error {
	return errors.ErrUnsupported
}

func IsExFATAvailable() bool {
	return false
}

func MakeExFAT(device string, label string) error {
	return errors.ErrUnsupported
}

func IsNTFSAvailable() bool {
	return false
}

func MakeNTFS(device string, label string) error {
	return errors.ErrUnsupported
}
