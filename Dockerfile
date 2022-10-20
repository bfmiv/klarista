FROM golang:1.19-buster as dev

# Download packr binary
RUN go install github.com/gobuffalo/packr/v2/packr2@v2.8.3

WORKDIR $GOPATH/src/github.com/bfmiv/klarista

COPY go.mod go.sum ./

RUN go mod download

COPY . .

FROM golang:1.19-buster as build

ARG KLARISTA_CLI_VERSION
ENV KLARISTA_CLI_VERSION ${KLARISTA_CLI_VERSION}

COPY --from=dev /go /go
WORKDIR $GOPATH/src/github.com/bfmiv/klarista
RUN ./scripts/build.sh

FROM debian:buster

ARG KLARISTA_CLI_VERSION
ARG BUILD_STAGE_WORKDIR=/go/src/github.com/bfmiv/klarista

COPY --from=build ${BUILD_STAGE_WORKDIR}/scripts/install.sh /usr/local/bin/install
COPY --from=build ${BUILD_STAGE_WORKDIR}/bin/* /

RUN ln -s /klarista-linux-amd64 /usr/local/bin/klarista
