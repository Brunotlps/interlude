package proxy

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

type Proxy struct {
	target *url.URL
	client *http.Client
}

func New(backendURL string) (*Proxy, error) {

	target, err := url.Parse(backendURL)
	if err != nil {
		return nil, fmt.Errorf("parsing backend URL %q: %w", backendURL, err)
	}
	return &Proxy{
		target: target,
		client: &http.Client{Timeout: 30 * time.Second},
	}, nil

}

func (p *Proxy) Do(r *http.Request) (*http.Response, error) {
	out, err := http.NewRequestWithContext(r.Context(), r.Method, p.target.String()+r.URL.RequestURI(), r.Body)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}

	for k, v := range r.Header {
		out.Header[k] = v
	}
	out.Header.Set("X-Forwarded-For", r.RemoteAddr)
	out.Header.Set("X-Gateway", "interlude")

	return p.client.Do(out)
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	resp, err := p.Do(r)
	if err != nil {
		slog.Error("proxy: backend unreachable", "backend", p.target.Host, "err", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprintf(w, `{"error": "backend unavailable"}`)
		return
	}
	defer resp.Body.Close()

	for k, v := range resp.Header {
		for _, vv := range v {
			w.Header().Add(k, vv)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}
