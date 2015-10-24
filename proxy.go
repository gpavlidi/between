package between

import (
	"log"
	"net/http"
	"net/http/httputil"
)

type Proxy struct {
	ProcessRequest  func(req *http.Request) error
	ProcessResponse func(resp *http.Response) error
	HttpProxy       *httputil.ReverseProxy
	HttpsProxy      *httputil.ReverseProxy
	http.RoundTripper
}

func NewProxy() *Proxy {
	proxy := &Proxy{}

	// proxyDirectors are used to intercept requests
	httpDirector := proxy.proxyDirector("http")
	httpsDirector := proxy.proxyDirector("https")

	// RoundTrip is used to intercept responses by http.DefaultTransport
	proxy.RoundTripper = http.DefaultTransport

	// set default request/response handlers
	proxy.ProcessRequest = defaultProcessRequest
	proxy.ProcessResponse = defaultProcessResponse

	// use native httputil.ReverseProxy, it ROCKS!
	proxy.HttpProxy = &httputil.ReverseProxy{Director: httpDirector, Transport: proxy}
	proxy.HttpsProxy = &httputil.ReverseProxy{Director: httpsDirector, Transport: proxy}

	return proxy
}

// Director is used to intercept requests
func (p *Proxy) proxyDirector(scheme string) func(req *http.Request) {
	return func(req *http.Request) {
		req.URL.Scheme = scheme
		req.URL.Host = req.Host
		p.ProcessRequest(req)
	}
}

// RoundTrip is used to intercept responses by http.DefaultTransport
func (p *Proxy) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	resp, err = p.RoundTripper.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	p.ProcessResponse(resp)
	return resp, nil
}

func defaultProcessRequest(req *http.Request) error {
	log.Println("Request: ", req.Method, req.URL.String())
	return nil
}

func defaultProcessResponse(resp *http.Response) error {
	log.Println("Response: ", resp.Status, resp.ContentLength)
	return nil
}
