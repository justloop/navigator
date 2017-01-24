package hashring

import "errors"

// All the errors related to hashring
var (
	ErrGetKeyNode  = errors.New("Ring return error in GetKeyNode")
	ErrGetKeyNodes = errors.New("Ring return error in get Node")
)
