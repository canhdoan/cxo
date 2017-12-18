package skyobject

import (
	"errors"

	"github.com/skycoin/skycoin/src/cipher"
)

// common errors
var (
	ErrRootIsHeld       = errors.New("Root is held")
	ErrRootIsNotHeld    = errors.New("Root is not held")
	ErrObjectIsTooLarge = errors.New("object is too large (see MaxObjectSize)")
	ErrTerminated       = errors.New("terminated")
)

// ObjectIsTooLargeError represents error that
// occurs when an object exceed max object size
// limit. The error contains hash of the object
type ObjectIsTooLargeError struct {
	hash cipher.SHA256
}

// Hash of the large object
func (o *ObjectIsTooLargeError) Hash() cipher.SHA256 {
	return o.hash
}

// Error implements error interface
func (o *ObjectIsTooLargeError) Error() string {
	return "object is too large: " + o.Hash().Hex()[:7]
}
