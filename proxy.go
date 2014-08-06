// Copyright 2014 Justin Wilson. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package proxy is used to reverse proxy requests to Docker containers.
//
// An example use is:
//
//  package main
//
//  import (
//  	"net/http"
//
//  	"code.minty.io/docker-proxy"
//  	"code.minty.io/marbles/mux"
//  )
//
//  func main() {
//	 	m := mux.NewServeMux()
//  	if err := proxy.SetupRoutes(m); err != nil {
//  		panic(err)
//  	}
//
//  	// API
//  	go http.ListenAndServe(":7824", proxy.APIHandler(m))
//  	// Proxy
//  	go http.ListenAndServe(":8080", m)
//  	http.ListenAndServeTLS(":8443", "server.crt", "server.key", m)
//  }
//
// % docker run -it --rm -p 80:8080 -p 443:8443 -p 7824:7824 -e "DOCKER_HOST=http://172.16.120.120:4243" -v "`pwd`":/opt/go -w /opt/go dev-go /bin/bash

package proxy

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"code.minty.io/roxy"
	"github.com/samalba/dockerclient"
)

// Type Proxy implements an http.Handle
// that acts as a proxy for Docekr containers.
type Proxy struct {
	*roxy.Proxy
}

// Type Route holds basic information about a container.
type Route struct {
	Host, Path string
	IP         string
	Port       int
	created    int
}

// Gets an environment variables value.
func envVariable(key string, env []string) (string, bool) {
	key = key + "="
	l := len(key)
	for _, val := range env {
		if len(val) < 10 || val[:l] != key {
			continue
		}
		return val[l:], true
	}
	return "", false
}

// A director for proxying to a containers path.
func toHost(host string) func(*http.Request) {
	return func(r *http.Request) {
		r.URL.Scheme = "http"
		r.URL.Host = host
	}
}

// Returns a new Docker client.
func newDockerClient(host string) (*dockerclient.DockerClient, error) {
	if len(host) > 3 && host[:3] == "tcp" {
		host = "http" + host[3:]
	}
	return dockerclient.NewDockerClient(host)
}

// Generates a map of routes for all running containers that have an `ENDPOINT`
// environment variable set.
func RoutesFromHost(host string) (map[string]Route, error) {
	docker, err := newDockerClient(host)
	if err != nil {
		return nil, err
	}

	containers, err := docker.ListContainers(false)
	if err != nil {
		return nil, err
	}

	routes := make(map[string]Route)
	for _, c := range containers {
		ci, err := docker.InspectContainer(c.Id)
		if err != nil {
			return nil, err
		}

		path, pathOk := envVariable("ENDPOINT", ci.Config.Env)
		host, hostOk := envVariable("HOST", ci.Config.Env)
		if pathOk == hostOk { // Hackety XOR.
			continue
		}

		p, ok := envVariable("TCP_PORT", ci.Config.Env)
		if !ok {
			p = "8080"
		}
		port, err := strconv.Atoi(p)
		if err != nil {
			port = 8080
		}

		n := ci.NetworkSettings
		key := fmt.Sprintf("%s:%s", host, path)
		if r, ok := routes[key]; ok {
			if r.created > c.Created {
				continue
			}
		}
		routes[key] = Route{host, path, n.IpAddress, port, c.Created}
	}
	return routes, err
}

// Adds all container routes to the given mux.
// Running this after routes have already been setup will do a refresh/reload.
func SetupRoutes(p *Proxy) error {
	routes, err := RoutesFromHost(os.Getenv("DOCKER_HOST"))
	if err != nil {
		return err
	}

	p.ClearHosts()
	p.ClearPatterns()
	for _, route := range routes {
		host := fmt.Sprintf("%s:%d", route.IP, route.Port)
		if route.Path != "" {
			log.Printf("proxying path '%s' to '%s'\n", route.Path, host)
			p.ForPattern(route.Path, roxy.ToHost(host))
		} else {
			log.Printf("proxying host '%s' to '%s'\n", route.Host, host)
			p.ForHost(route.Host, roxy.ToHost(host))
		}
	}
	return nil
}

// Simple API handler (currently only supports POST with a query-string of `?action=reload`).
func APIHandler(p *Proxy) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		switch r.URL.Query().Get("action") {
		default:
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Invalid, or missing, 'action'"))
		case "reload":
			log.Println("reloading routes")
			SetupRoutes(p)
		}
	})
}

// New returns a new Proxy.
func New() *Proxy {
	return &Proxy{roxy.New()}
}
