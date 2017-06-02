package main

import (
	"io/ioutil"
	"regexp"

	"github.com/mikegleasonjr/devproxy"
	"gopkg.in/yaml.v2"
)

type config struct {
	Bind  string
	Port  uint
	Debug bool
	Hosts []*spoof
}

func (c *config) HostsSpoofs() []devproxy.Spoofer {
	s := make([]devproxy.Spoofer, len(c.Hosts))
	for i, h := range c.Hosts {
		s[i] = devproxy.Spoofer(h)
	}
	return s
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

func (s *spoof) UnmarshalYAML(unmarshal func(interface{}) error) (err error) {
	var enveloppe map[string]string
	err = unmarshal(&enveloppe)
	if err != nil {
		return
	}
	for f, r := range enveloppe {
		s.m, err = regexp.Compile(f)
		s.r = r
	}
	return
}

func loadConfig(files ...string) (c config, err error) {
	var b []byte

	for _, f := range files {
		b, err = ioutil.ReadFile(f)
		if err == nil {
			break
		}
	}

	if err != nil {
		return
	}

	err = yaml.Unmarshal(b, &c)
	return
}
