package imprint

import "errors"

// Source: https://github.com/retrixe/imprint/blob/main/app/devices.go

// ErrNotBlockDevice is returned when the specified device is not a block device.
var ErrNotBlockDevice = errors.New("specified device is not a block device")
