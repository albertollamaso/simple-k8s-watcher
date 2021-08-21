FROM golang:1.14

WORKDIR $GOPATH/src/github.com/albertollamaso/simple-k8s-watcher

COPY . .

RUN go mod init

RUN go build

RUN go get -d -v ./...

RUN go install -v ./...

CMD ["simple-k8s-watcher"]