package proxy

import (
	"net/http/httputil"
	"net/url"
)

func NewProxy(target string) *httputil.ReverseProxy {
	parsedURL, _ := url.Parse(target)
	return httputil.NewSingleHostReverseProxy(parsedURL)
}
