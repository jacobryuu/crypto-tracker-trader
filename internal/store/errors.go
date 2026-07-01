package store

import "errors"

// ErrNotFound is returned when a requested record does not exist
// or does not belong to the requesting user.
var ErrNotFound = errors.New("record not found")
