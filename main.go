package main

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/elazarl/goproxy"
	"github.com/joeshaw/envdecode"
	"go.uber.org/dig"
	"log"
	"net/http"
	"strings"
)

func provide() *dig.Container {
	c := dig.New()
	c.Provide(NewConfig)
	c.Provide(NewProxy)
	c.Provide(NewDocker)
	c.Provide(MatchHook)

	return c
}

func main() {
	provide().Invoke(func(cfg *Config, proxy *goproxy.ProxyHttpServer) {
		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", cfg.Port), proxy))
	})
}

type Config struct {
	Hostname string `env:"HOSTNAME"`
	Port     int    `env:"PORT,default=8080"`
}

func NewConfig() (*Config, error) {
	cfg := &Config{}
	err := envdecode.Decode(cfg)

	return cfg, err
}

type Docker struct {
	*client.Client
}

func (d *Docker) Find(host string, hostname string, ctx *goproxy.ProxyCtx) (bool, string) {
	args := filters.NewArgs()
	args.Add("label", fmt.Sprintf("vhost=%s", host))

	containers, err := d.ContainerList(context.Background(), types.ContainerListOptions{Filters: args})
	if err != nil {
		ctx.Logf("Error: Container list: %s", err.Error())
	}

	for _, container := range containers {
		data, err := d.ContainerInspect(context.Background(), container.ID)

		if err != nil {
			ctx.Logf("Error: Container inspect: %s", err.Error())
		}

		network := "bridge"
		if !data.HostConfig.NetworkMode.IsDefault() {
			network = data.HostConfig.NetworkMode.NetworkName()

			if hostname != "" {
				d.NetworkConnect(context.Background(), container.NetworkSettings.Networks[network].NetworkID, hostname, nil)
			}
		}

		ip := container.NetworkSettings.Networks[network].IPAddress

		return true, ip
	}

	return false, ""
}

func NewDocker() (*Docker, error) {
	cli, err := client.NewEnvClient()
	if err != nil {
		return nil, err
	}

	return &Docker{cli}, nil
}

func NewProxy(hook func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response)) *goproxy.ProxyHttpServer {
	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = true
	proxy.OnRequest().DoFunc(hook)

	return proxy
}

func MatchHook(cfg *Config, d *Docker) func(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
	return func(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		ctx.Logf("Searching in docker: %s", r.URL.Hostname())

		if ok, host := d.Find(r.URL.Hostname(), cfg.Hostname, ctx); ok {
			r.Host = strings.Join([]string{host, r.URL.Port()}, ":")
			r.URL.Host = r.Host

			ctx.Logf("Request has been moved to %s", host)

			return r, nil
		}

		ctx.Logf("VHost %s doesn't exist in docker", r.Host)

		return r, nil
	}
}
