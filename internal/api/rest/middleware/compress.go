// Package middleware provides various middleware functionality.
package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

// Type gzipWriter redefines http.ResponseWriter changing its Writer method to use gzip.
type gzipWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

// Write method redefines default http.ResponseWriter Write method.
func (w gzipWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

// CompressHandle serves as a middleware handler implementing gzip compressing.
func CompressHandle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
		if err != nil {
			io.WriteString(w, err.Error())
			return
		}
		defer gz.Close()
		w.Header().Set("Content-Encoding", "gzip")
		next.ServeHTTP(gzipWriter{ResponseWriter: w, Writer: gz}, r)
	})
}

// DecompressHandle serves as a middleware handler implementing gzip decompressing.
func DecompressHandle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		gz, err := gzip.NewReader(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		r.Body = gz
		next.ServeHTTP(w, r)
	})
}
