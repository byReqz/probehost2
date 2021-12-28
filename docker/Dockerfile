FROM alpine:latest

RUN apk update
RUN apk add curl jq 
RUN curl -s https://api.github.com/repos/byReqz/probehost2/releases/latest | jq -r '.assets[].browser_download_url' | grep "probehost2" | xargs curl -L -o probehost2 
ENTRYPOINT ["/probehost2"]
