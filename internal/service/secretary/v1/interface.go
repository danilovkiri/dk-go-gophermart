// Package secretary provides methods for ciphering.
package secretary

import "net/http"

// Secretary defines a set of methods for types implementing Secretary.
type Secretary interface {
	Encode(data string) string
	Decode(msg string) (string, error)
	NewCookie() (*http.Cookie, string)
}
