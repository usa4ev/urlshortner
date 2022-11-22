package storageerrors

import "errors"

var (
	ErrConflict = errors.New("URL has already been shortened")
	ErrURLGone  = errors.New("URL with this id is deleted")
)
