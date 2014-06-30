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
//  )
//
//  func main() {
//	 	m := mux.NewServeMux()
//  	if err := SetupRoutes(m); err != nil {
//  		panic(err)
//  	}
//
//  	// API
//  	go http.ListenAndServe(":7824", APIHandler(m))
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
	"net/http/httputil"
	"os"
	"strconv"

	"code.minty.io/marbles/mux"
	"github.com/samalba/dockerclient"
)

// Type Route holds basic information about a container.
type Route struct {
	Path    string
	IP      string
	Port    int
	created int
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

// Generates a map of routes for all running containers that have an `ENDPOINT`
// environment variable set.
func RoutesFromHost(host string) (map[string]Route, error) {
	docker, err := dockerclient.NewDockerClient(host)
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

		path, ok := envVariable("ENDPOINT", ci.Config.Env)
		if !ok {
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
		if r, ok := routes[path]; ok {
			if r.created > c.Created {
				continue
			}
		}
		routes[path] = Route{path, n.IpAddress, port, c.Created}
	}
	return routes, err
}

// Adds all container routes to the given mux.
// Running this after routes have already been setup will do a refresh/reload.
func SetupRoutes(m *mux.ServeMux) error {
	routes, err := RoutesFromHost(os.Getenv("DOCKER_HOST"))
	if err != nil {
		return err
	}

	for path, route := range routes {
		host := fmt.Sprintf("%s:%d", route.IP, route.Port)
		log.Printf("proxying path '%s' to '%s'\n", path, host)
		m.UnHandle(path)
		m.Handle(path, &httputil.ReverseProxy{Director: toHost(host)})
	}
	return nil
}

// Simple API handler (currently only supports POST with a query-string of `?action=reload`)
func APIHandler(m *mux.ServeMux) http.Handler {
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
			SetupRoutes(m)
		}
	})
}
