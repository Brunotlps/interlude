package proxy

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

// Create an http.Handler that works as reverse proxy for the informed backend
func New(backendURL string) (http.Handler, error) {

	// Converts a string into a url.URL struct
	// Validates the URL and separates scheme, host, port, path,...
	target, err := url.Parse(backendURL)
	if err != nil {
		return nil, fmt.Errorf("parsing backend URL %q: %w", backendURL, err)
	}

	proxy := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {

			r.SetURL(target)                           // Defines the target schema, host and path
			r.Out.Host = r.In.Host                     // Preserves the original host
			r.SetXForwarded()                          // Correctly sets X-Forwarded-For, X-Forwarded-Host and X-Forwarded-Proto safely
			r.Out.Header.Set("X-Gateway", "interlude") // Adds a custom header to the request that will be sent to the backend

		},
	}

	// Error Handler in case of backend failure
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("[ERROR] Failed proxy to: %s: %v", backendURL, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(`{"error": "backend unavailable", "status": 502}`))
	}

	return proxy, nil

}
