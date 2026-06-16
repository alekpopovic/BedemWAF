package auth

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

type StaticBearer struct {
	token string
}

func NewStaticBearer(token string) StaticBearer {
	return StaticBearer{token: token}
}

func (a StaticBearer) Enabled() bool {
	return a.token != ""
}

func (a StaticBearer) Authorized(r *http.Request) bool {
	if a.token == "" {
		return false
	}
	const prefix = "Bearer "
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, prefix) {
		return false
	}
	got := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	return subtle.ConstantTimeCompare([]byte(got), []byte(a.token)) == 1
}
