package response

import (
	"net/http"
)

// Unauthorized writes the unauthorized response.
func Unauthorized(w http.ResponseWriter) {
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte("Unauthorized"))
}
