package devproxy

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"testing"
)

func TestProxy(t *testing.T) {
	var xff string

	p, c := testProxy()
	o := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		xff = req.Header.Get("X-Forwarded-For")
	}))

	defer p.Close()
	defer o.Close()

	res, err := c.Get(o.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if xff != "127.0.0.1" {
		t.Errorf("expected X-Forwarded-For header to be present: expected '%s', got '%s'", "127.0.0.1", xff)
	}
}

func TestConnect(t *testing.T) {
	p, _ := testProxy()
	o := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("OK!"))
	}))

	defer p.Close()
	defer o.Close()

	proxyURL, _ := url.Parse(p.URL)
	originURL, _ := url.Parse(o.URL)

	c, err := net.Dial("tcp", proxyURL.Host)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	_, err = c.Write([]byte(fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n", originURL.Host, originURL.Host)))
	if err != nil {
		t.Fatal(err)
	}

	b := make([]byte, 16*1024)
	n, err := c.Read(b)
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Compare(b[0:n], connEstablishedReply) != 0 {
		t.Fatalf("invalid response received with CONNECT method: expected '%s', got '%s'", connEstablishedReply, b[0:n])
	}

	_, err = c.Write([]byte(fmt.Sprintf("GET / HTTP/1.1\r\nHost: %s\r\n\r\n", originURL.Host)))
	if err != nil {
		t.Fatal(err)
	}

	n, err = c.Read(b)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Contains(b[0:n], []byte("OK!")) {
		t.Fatalf("invalid response received from origin: expected response to contain '%s', got '%s'", "OK!", b[0:n])
	}
}

func TestHostSpoof(t *testing.T) {
	o := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("OK!"))
	}))
	originURL, _ := url.Parse(o.URL)
	p, c := testProxy(WithHosts([]Spoofer{
		&spoof{regexp.MustCompile(`^test\.com:80$`), originURL.Host},
	}))

	defer p.Close()
	defer o.Close()

	res, err := c.Get("http://test.com/")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	b, _ := ioutil.ReadAll(res.Body)
	if bytes.Compare(b, []byte("OK!")) != 0 {
		t.Fatalf("invalid response received from origin: expected response to contain '%s', got '%s'", "OK!", b)
	}
}

func testProxy(opts ...option) (*httptest.Server, *http.Client) {
	p := httptest.NewServer(New(opts...))
	u, _ := url.Parse(p.URL)
	t := &http.Transport{Proxy: http.ProxyURL(u)}
	c := &http.Client{Transport: t}
	return p, c
}

type spoof struct {
	m *regexp.Regexp
	r string
}

func (s *spoof) Match(str string) bool {
	return s.m.MatchString(str)
}

func (s *spoof) Replace(str string) string {
	return s.m.ReplaceAllString(str, s.r)
}
