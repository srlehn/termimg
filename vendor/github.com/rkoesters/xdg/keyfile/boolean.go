package keyfile

import (
	"strconv"
)

// Bool returns the value for group 'g' and key 'k' as a bool.
func (kf *KeyFile) Bool(g, k string) (bool, error) {
	return strconv.ParseBool(kf.Value(g, k))
}
