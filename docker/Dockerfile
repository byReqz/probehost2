FROM golang:latest as builder
WORKDIR /build
COPY . .
RUN go get -u
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o probehost2

FROM alpine:latest
RUN apk update
RUN apk add mtr iputils nmap-nping traceroute
RUN adduser -D probehost2
COPY --from=builder /build/probehost2 /
RUN touch /probehost2.log
RUN chown probehost2:users /probehost2.log
USER probehost2
CMD ["/probehost2"]
