package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/user"
	"path/filepath"

	"github.com/mikegleasonjr/devproxy"
	"gopkg.in/urfave/cli.v2"
)

const (
	version = "0.0.1"
)

// TODO, auto-configure on different platforms:
//
// osx:
// networksetup -setwebproxy "<conn id>" localhost <port>
// networksetup -setwebproxystate "<conn id>" off

func main() {
	app := &cli.App{
		Name:     "devproxy",
		Usage:    "a local development proxy",
		Version:  version,
		Commands: []*cli.Command{},
		Flags: []cli.Flag{
			&cli.UintFlag{
				Name:    "port",
				Value:   8080,
				Usage:   "listening port",
				Aliases: []string{"p"},
			},
			&cli.StringFlag{
				Name:    "bind",
				Value:   "localhost",
				Usage:   "listening interface",
				Aliases: []string{"b"},
			},
			&cli.StringFlag{
				Name:        "config",
				Value:       ".devproxy.yml",
				Usage:       "config file",
				Aliases:     []string{"c"},
				DefaultText: "first of .devproxy.yml, ~/.devproxy.yml",
			},
			&cli.BoolFlag{
				Name:    "debug",
				Value:   false,
				Usage:   "print requests to stdout",
				Aliases: []string{"d"},
			},
		},
		Action: func(c *cli.Context) error {
			configs := []string{".devproxy.yml"}

			if specified(c, "config") {
				configs = []string{c.String("config")}
			} else if u, err := user.Current(); err == nil {
				configs = append(configs, filepath.Join(u.HomeDir, configs[0]))
			}

			conf, err := loadConfig(configs...)
			if err != nil {
				return err
			}

			if specified(c, "bind") {
				conf.Bind = c.String("bind")
			}

			if specified(c, "port") {
				conf.Port = c.Uint("port")
			}

			if specified(c, "debug") {
				conf.Debug = c.Bool("debug")
			}

			proxy := devproxy.New(
				devproxy.WithHosts(conf.HostsSpoofs()),
				devproxy.WithDebugOutput(conf.Debug),
			)

			listen := fmt.Sprintf("%s:%d", conf.Bind, conf.Port)
			log.SetPrefix("[devproxy] ")
			log.Println("starting on", listen)
			return http.ListenAndServe(listen, proxy)
		}}

	app.Run(os.Args)
}

func specified(c *cli.Context, flag string) bool {
	for _, f := range c.FlagNames() {
		if f == flag {
			return true
		}
	}
	return false
}
