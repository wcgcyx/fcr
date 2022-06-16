FROM golang:1.17.9-alpine

RUN apk update && apk add --no-cache make gcc musl-dev linux-headers git
WORKDIR /app/
COPY . .
RUN make build && go build -o ./build/mc itest/mc/* && mv build/* $GOPATH/bin/
RUN mkdir ~/.fcr/
