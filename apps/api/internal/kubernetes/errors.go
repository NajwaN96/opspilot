package kubernetes

import "errors"

// ErrDisconnected is returned when a caller asks for an object and no API is configured.
var ErrDisconnected = errors.New("kubernetes cluster is disconnected")
