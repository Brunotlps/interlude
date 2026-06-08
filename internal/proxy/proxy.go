package proxy

import (
	"fmt"
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
