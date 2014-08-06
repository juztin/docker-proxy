# Docker Proxy                                                                                                                                                                                                                                                    
#
# VERSION   1
#
# docker run -it --rm -p 80:8080 -p 443:8443 -p 7824:7824 -e "DOCKER_HOST=unix:///var/run/docker.sock" -v /var/run/docker.sock:/var/run/docker.sock juztin/docker-proxy
#
# docker run -it --rm -p 80:8080 -p 443:8443 -p 7824:7824 -e "DOCKER_HOST=unix:///var/run/docker.sock" juztin/docker-proxy
# docker run -it --rm -p 80:8080 -p 443:8443 -p 7824:7824 -e "DOCKER_HOST=http://172.16.120.120:4243" juztin/docker-proxy
#
# docker run -it --rm \
#    -p 7824:7824 \
#    -p 80:8080 \
#    -p 443:8443 \
#    -e "DOCKER_HOST=unix:///var/run/docker.sock" \
#    -v /var/run/docker.sock:/var/run/docker.sock \
#    -w /opt/go stampinup/dev-go \
#    juztinwilzon/docker-proxy
#
#    docker run -it --rm -p 7824:7824 -p 80:8080 -p 443:8443 -e "DOCKER_HOST=unix:///var/run/docker.sock" -v /var/run/docker.sock:/var/run/docker.sock juztinwilzon/docker-proxy
#    docker run -it --rm -p 7824:7824 -p 80:8080 -p 443:8443 -e "TLS_KEY=/opt/proxy/scribbled.key" -e "TLS_CERT=/opt/proxy/scribbled.crt" -e "DOCKER_HOST=unix:///var/run/docker.sock" -v /var/run/docker.sock:/var/run/docker.sock juztinwilzon/docker-proxy
#


FROM ubuntu:14.04


MAINTAINER Justin Wilson justin@minty.io


ADD ./docker-proxy /opt/docker-proxy/


# Update
RUN apt-get update
RUN apt-get install -y ca-certificates


EXPOSE 8080
EXPOSE 8443
EXPOSE 7824


WORKDIR /opt/docker-proxy
ENTRYPOINT ./docker-proxy
