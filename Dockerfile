# base image
FROM golang:1.24.2-alpine as base
WORKDIR /flowx

ENV CGO_ENABLED=0

COPY go.mod go.sum /flowx/
RUN go mod download

ADD . .
RUN go build -o /usr/local/bin/flowx ./cmd/flowx

# runner image with shell (alpine)
FROM alpine:latest
RUN apk add --no-cache tzdata curl

WORKDIR /app
COPY --from=base /usr/local/bin/flowx flowx

EXPOSE 3625
ENTRYPOINT ["/app/flowx"]
