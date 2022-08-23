package storageerrors

import "errors"

var ErrConflict = errors.New("URL has already been shortened")
