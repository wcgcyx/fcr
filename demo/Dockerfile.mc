FROM golang:1.16-alpine

RUN apk update && apk add --no-cache make gcc musl-dev linux-headers git
WORKDIR /app/
COPY . .
RUN make mc

CMD [ "./build/mc" ]
EXPOSE 9010
