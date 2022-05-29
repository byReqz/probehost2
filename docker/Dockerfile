FROM golang:latest as builder
WORKDIR /build
COPY . .
RUN go get -u
RUN CGO_ENABLED=0 go build -o probehost2

FROM alpine:latest
RUN apk update
RUN apk add mtr iputils nmap-nping
COPY --from=builder /build/probehost2 /
RUN touch /probehost2.log
CMD ["/probehost2"]
