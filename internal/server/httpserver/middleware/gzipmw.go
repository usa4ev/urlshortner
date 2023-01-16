package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
)

// gzipWriter is used to replace default http.ResponseWriter
// when handling requests fom clients that support compressed encoding.
type gzipWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (w gzipWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}


// GzipMW middleware replaces default http.ResponseWriter with
// gzipWriter if the client supports gzip encoding.
func GzipMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		supportedEncoding := r.Header.Values("Accept-Encoding")
		if len(supportedEncoding) > 0 {
			for _, v := range supportedEncoding {
				if v == "gzip" {
					writer, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)

						return
					}
					defer writer.Close()
					w.Header().Set("Content-Encoding", "gzip")
					gzipW := gzipWriter{w, writer}
					next.ServeHTTP(gzipW, r)

					return
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}
