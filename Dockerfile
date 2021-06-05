FROM golang:1.16
WORKDIR /go/src/github.com/webuild-community/core
COPY . .
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o server cmd/*.go

FROM alpine:latest  
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=0 /go/src/github.com/webuild-community/core/server .
CMD ["./server"]  