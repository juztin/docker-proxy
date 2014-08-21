package main

import (
	"log"
	"net/http"
	"os"

	"code.minty.io/docker-proxy"
)

/*
    --link "service-users:service-users" \
    -e "DOCKER_HOST=http://172.16.120.120:4243" \
    -v "`pwd`":/opt/go \

docker run -it --rm \
    -p 7824:7824 \
    -p 80:8080 \
    -p 443:8443 \
    -e "DOCKER_HOST=unix:///var/run/docker.sock" \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -w /opt/go stampinup/dev-go \
    juztinwilzon/docker-proxy

    docker run -it --rm -p 7824:7824 -p 80:8080 -p 443:8443 -e "DOCKER_HOST=unix:///var/run/docker.sock" -v /var/run/docker.sock juztinwilzon/docker-proxy
    docker run -it --rm -p 7824:7824 -p 80:8080 -p 443:8443 -e "TLS_KEY=/opt/proxy/scribbled.key" -e "TLS_CERT=/opt/proxy/scribbled.crt" -e "DOCKER_HOST=unix:///var/run/docker.sock" -v /var/run/docker.sock juztinwilzon/docker-proxy
*/

func main() {
	p := proxy.New()
	if err := proxy.SetupRoutes(p); err != nil {
		log.Fatalln(err)
	}

	// API
	log.Println("API listening on 7824")
	go http.ListenAndServe(":7824", proxy.APIHandler(p))

	// Proxy
	key, cert, hasCerts := cert()
	if !hasCerts {
		log.Printf("Certificate missing, key: '%s', cert: '%s'", key, cert)
		log.Println("Listening on 8080")
		http.ListenAndServe(":8080", p)
	} else {
		log.Println("Listening on 8080, 8443")
		go http.ListenAndServe(":8080", p)
		http.ListenAndServeTLS(":8443", cert, key, p)
	}
}

func cert() (key string, cert string, ok bool) {
	ok = true
	key = os.Getenv("TLS_KEY")
	cert = os.Getenv("TLS_CERT")
	if key == "" {
		key = "/opt/proxy/proxy.key"
	}
	if cert == "" {
		cert = "/opt/proxy/proxy.crt"
	}
	if _, err := os.Stat(key); err != nil {
		ok = false
	}
	if _, err := os.Stat(cert); err != nil {
		ok = false
	}
	return
}
