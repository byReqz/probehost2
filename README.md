# probehost2
an http endpoint to query network diagnosis tools from remote hosts

1. Overview
2. Disclaimer
3. Installation
4. Usage

# Disclaimer
Dont expect good or even mediocre code here. This is my first take at go and is mostly for myself to learn. Suggestions and improvements are welcome.

Please note that this project does not include any kind of rate limiting or other protection. It is therefore heavily advised to only make it publicly reachable if a reverse proxy is in place. A sample config for <a href="caddyserver.com/">Caddy</a> can be found in the `caddy` subfolder. 

# Installation
The runtime dependencies are currently `iputils`, `traceroute` and `mtr` (sometimes called `mtr-tiny`). `iputils` and `traceroute` can be substituted by `busybox`.

## Building
The app can be built with the latest Go toolchain.

First get the external module dependencies:
```sh
go get -u
```
Then build the app, this already strips debug symbols. 
```sh
go build -ldflags "-s -w" -o "probehost2" main.go
```
(if this is unwanted, just leave out the ldflags argument)

## Systemd
Example files for a systemd service can be found in the `systemd` subfolder.

## Docker
A docker container based on <a href="https://alpinelinux.org">Alpine</a> can be built by using the included dockerfile (`docker/Dockerfile`).
```sh
docker build -f docker/Dockerfile . -t byreqz/probehost2:latest
```
A compose file can also be found in `docker/docker-compose.yml`.

## Proxy
Its recommended to only run this app together with a rate-limiting reverse-proxy. An example configuration for <a href="caddyserver.com/">Caddy</a> can be found in the `caddy` subfolder. 

# Usage
## Server
The app currently has 4 runtime flags:
- `-p / --listenport` -- sets the port to listen on
- `-o / --logfilepath` -- sets the log output file
- `-x / --disable-x-forwarded-for` -- disables checking for the X-Forwarded-For header
- `-l / --allow-private` -- allows lookups of private IP ranges

The app will log every request including the IP thats querying and show failed requests on stdout.

## Client
### General
The app can be queried via HTTP/HTTPS with the following scheme:
```
https://[address]/[command]/[hosts]/[options]
```

- [address] = the IP or domain serving the site
- [command] = the command to return, currently available:
  - ping
  - mtr
  - traceroute
- [hosts] = can be one or more hosts query, seperated by a comma
- [options] = options to run the command with, seperated by a comma

All inputs are validated and invalid input is discarded. If the request contains no valid data, the server will return HTTP 500.

Local IP ranges are by default excluded from lookups, this currently only includes IPs and not hostnames and can be disabled on the server by passing the -l flag.

Command options are based on the originally given cli flags but also have a more understandable altname (wip).

### Ping
The default options are:
- `-c 10`: send 10 pings

Available options are:
- `4`: force IPv4
- `6`: force IPv6
- `d`: print timestamps
- `n`: no dns name resolution
- `v`: verbose output
- `c1`: send 1 ping
- `c5`: send 5 pings
- `c10`: send 10 pings

Example query:
```sh
$ curl http://localhost:8000/ping/localhost/c1
PING localhost(localhost (::1)) 56 data bytes
64 bytes from localhost (::1): icmp_seq=1 ttl=64 time=0.189 ms

--- localhost ping statistics ---
1 packets transmitted, 1 received, 0% packet loss, time 0ms
rtt min/avg/max/mdev = 0.189/0.189/0.189/0.000 ms
```

### MTR
The default options are:
- `-r`: output using report mode
- `-w`: output wide report
- `-c10`: send 10 pings

Available options are:
- `4`: force IPv4
- `6`: force IPv6
- `u`: use UDP instead of ICMP echo
- `t`: use TCP instead of ICMP echo
- `e`: display information from ICMP extensions
- `x`: output xml
- `n`: do not resolve host names
- `b`: show IP numbers and host names
- `z`: display AS number
- `c1`: send 1 ping
- `c5`: send 5 pings
- `c10`: send 10 pings

Example query:
```
$ curl http://localhost:8000/mtr/localhost/c1,z
Start: 2022-01-02T00:06:56+0100
HOST: xxx                 Loss%   Snt   Last   Avg  Best  Wrst StDev
  1. AS???    localhost   0.0%     1    0.6   0.6   0.6   0.6   0.0
```

### Traceroute
The default options are:
- none

Available options are:
- `4`: force IPv4
- `6`: force IPv6
- `f`: do not fragment packets
- `i`: use ICMP ECHO for tracerouting
- `t`: use TCP SYN for tracerouting (default port is 80)
- `n`: do not resolve IP addresses to their domain names
- `u`: use UDP to particular port for tracerouting (default port is 53)
- `ul`: Use UDPLITE for tracerouting (default port is 53)
- `d`: Use DCCP Request for tracerouting (default port is 33434)
- `b`: Guess the number of hops in the backward path and print if it differs

Example query:
```
$ curl http://localhost:8000/tracert/localhost/i 
traceroute to localhost (127.0.0.1), 30 hops max, 60 byte packets
 1  localhost (127.0.0.1)  0.063 ms  0.008 ms  0.006 ms

```