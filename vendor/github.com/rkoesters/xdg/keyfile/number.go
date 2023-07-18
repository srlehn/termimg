package keyfile

import (
	"strconv"
)

// Number returns the value for group 'g' and key 'k' as a float64.
func (kf *KeyFile) Number(g, k string) (float64, error) {
	return strconv.ParseFloat(kf.Value(g, k), 64)
}
