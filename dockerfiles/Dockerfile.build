FROM    golang:1.6-alpine

RUN     apk add -U git bash curl tree
RUN     export GLIDE=v0.12.0; \
        export SRC=https://github.com/Masterminds/glide/releases/download/; \
        curl -sL ${SRC}/${GLIDE}/glide-${GLIDE}-linux-amd64.tar.gz | \
        tar -xz linux-amd64/glide && \
        mv linux-amd64/glide /usr/bin/glide && \
        chmod +x /usr/bin/glide

RUN     go get github.com/dnephin/filewatcher && \
        cp /go/bin/filewatcher /usr/bin/ && \
        rm -rf /go/src/* /go/pkg/* /go/bin/*

RUN     go get github.com/jteeuwen/go-bindata/... && \
        cp /go/bin/go-bindata /usr/bin/ && \
        rm -rf /go/src/* /go/pkg/* /go/bin/*

WORKDIR /go/src/github.com/aanand/compose-file
ENV     PS1="# "
ENV     CGO_ENABLED=0
