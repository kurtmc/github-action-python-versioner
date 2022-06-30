FROM golang:1.18-alpine

RUN apk add --no-cache git
WORKDIR /app
COPY go.mod ./
COPY go.sum ./
RUN go mod download
COPY *.go ./
RUN go build -o /github-action-python-versioner
ENTRYPOINT ["/github-action-python-versioner"]
