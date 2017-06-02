# Introduction

[![GoDoc](https://godoc.org/github.com/mikegleasonjr/devproxy?status.svg)](https://godoc.org/github.com/mikegleasonjr/devproxy)

## Installation

```
go get github.com/mikegleasonjr/devproxy/cmd/devproxy
```

## Configuration example

By default, `devproxy` will look for a config file (`.devproxy.yml`) in the current working directory, then in the home folder.

```yaml
---
bind: 0.0.0.0
port: 8080
debug: true

hosts:
  - ^api\.website\.dev:80$: localhost:3000
  - ^golang\.dev:80$: localhost:6060
```

A request to `http://golang.dev/pkg` will proxy the request to `http://localhost:6060/pkg`.

## License

MIT
