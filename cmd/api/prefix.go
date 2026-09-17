package main

import (
	"context"
	"net/http"
	"strings"
)

type apiPrefixContextKey struct{}

// hadAPIPrefix reports whether stripAPIPrefix removed a leading "/api" from r,
// so handlers (e.g. a root redirect) can rebuild a correctly-prefixed Location.
func hadAPIPrefix(r *http.Request) bool {
	stripped, _ := r.Context().Value(apiPrefixContextKey{}).(bool)
	return stripped
}

// stripAPIPrefix strips a leading "/api" segment (e.g. from a CloudFront
// "/api/*" behavior) before dispatching to next, leaving other paths
// untouched so unprefixed requests (local Docker healthchecks, etc.) still work.
func stripAPIPrefix(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const prefix = "/api"

		if r.URL.Path == prefix {
			r.URL.Path = "/"
		} else if rest, ok := strings.CutPrefix(r.URL.Path, prefix+"/"); ok {
			r.URL.Path = "/" + rest
		} else {
			next.ServeHTTP(w, r)
			return
		}

		if r.URL.RawPath != "" {
			if r.URL.RawPath == prefix {
				r.URL.RawPath = "/"
			} else if rest, ok := strings.CutPrefix(r.URL.RawPath, prefix+"/"); ok {
				r.URL.RawPath = "/" + rest
			}
		}

		r = r.WithContext(context.WithValue(r.Context(), apiPrefixContextKey{}, true))
		next.ServeHTTP(w, r)
	})
}
