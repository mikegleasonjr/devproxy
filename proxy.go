package devproxy

import (
	"context"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	"sync"
)

var (
	connEstablishedReply = []byte("HTTP/1.1 200 Connection established\r\n\r\n")
)

// Proxy is the development proxy server
type Proxy struct {
	*httputil.ReverseProxy
	middlewares http.Handler
	logger      *log.Logger
	debug       bool
	hosts       []Spoofer
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	p.middlewares.ServeHTTP(w, req)
}

// Spoofer represent a spoofing rule.
type Spoofer interface {
	Match(string) bool
	Replace(string) string
}

type option func(*Proxy)

// WithDebugOutput prints each request headers to stdout
func WithDebugOutput(d bool) option {
	return func(p *Proxy) {
		p.debug = d
	}
}

// WithHosts configures which hosts to spoof
func WithHosts(h []Spoofer) option {
	return func(p *Proxy) {
		p.hosts = h
	}
}

// New creates a new development proxy
func New(opts ...option) *Proxy {
	director := func(req *http.Request) {
		if host, match := req.Context().Value(hostKeyMatch).(string); match {
			req.Host = host
			req.URL.Host = host
		}
	}

	p := &Proxy{
		logger: log.New(os.Stdout, "[devproxy] ", log.LstdFlags),
		ReverseProxy: &httputil.ReverseProxy{
			Director:   director,
			BufferPool: buffers,
			ErrorLog:   log.New(os.Stderr, "[devproxy] ", log.LstdFlags),
		},
	}

	for _, o := range opts {
		o(p)
	}

	p.middlewares = http.Handler(p.ReverseProxy)
	p.middlewares = connectMiddleware(p.middlewares)
	if p.debug {
		p.middlewares = debugMiddleware(p.logger, p.middlewares)
	}
	p.middlewares = detectMiddleware(p.hosts, p.middlewares)

	return p
}

type hostKey int

var hostKeyMatch hostKey = 1

func detectMiddleware(hosts []Spoofer, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		for _, s := range hosts {
			host := normalizeHost(req)
			if s.Match(host) {
				host = s.Replace(host)
				ctx := context.WithValue(req.Context(), hostKeyMatch, host)
				req = req.WithContext(ctx)
				break
			}
		}

		next.ServeHTTP(w, req)
	})
}

func debugMiddleware(log *log.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if host, match := req.Context().Value(hostKeyMatch).(string); match {
			log.Printf("Host spoof matched: %s -> %s\n", req.URL.Host, host)
		}

		b, err := httputil.DumpRequest(req, false)
		if err == nil {
			log.Printf("new request:\n%s", b)
		}

		next.ServeHTTP(w, req)
	})
}

func connectMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodConnect {
			if err := handleConnect(w, req); err != nil {
				http.Error(w, err.Error(), 500)
			}
			return
		}

		next.ServeHTTP(w, req)
	})
}

func handleConnect(w http.ResponseWriter, req *http.Request) error {
	// according to the RFC, the requested URL will always
	// have a port and has a form:
	// www.example.com:80
	// The request URL will then be: //www.example.com:80

	host, match := req.Context().Value(hostKeyMatch).(string)
	if !match {
		host = req.URL.Host
	}

	origin, err := net.Dial("tcp", host)
	if err != nil {
		return err
	}
	defer origin.Close()

	hj, ok := w.(http.Hijacker)
	if !ok {
		return errors.New("could not hijack connection")
	}

	client, _, err := hj.Hijack()
	if err != nil {
		return err
	}
	defer client.Close()

	client.Write(connEstablishedReply)

	close := sync.Once{}
	cp := func(dst io.WriteCloser, src io.ReadCloser) {
		b := buffers.Get()
		defer buffers.Put(b)
		io.CopyBuffer(dst, src, b)
		close.Do(func() {
			dst.Close()
			src.Close()
		})
	}
	go cp(origin, client)
	cp(client, origin)

	return nil
}

// normalizeHost will always return a host string
// of the form host:port for a giver request
func normalizeHost(req *http.Request) string {
	if hostHasPort(req.Host) {
		return req.Host
	}
	if hostHasPort(req.URL.Host) {
		return req.URL.Host
	}
	if req.URL.Scheme == "https" {
		return req.Host + ":443"
	}
	return req.Host + ":80"
}

func hostHasPort(host string) bool {
	return strings.LastIndex(host, ":") > strings.LastIndex(host, "]")
}
