FROM golang:1.22 as build-env

WORKDIR /go/src/webhookx-io/webhookx

COPY go.mod /go/src/webhookx-io/webhookx
COPY go.sum /go/src/webhookx-io/webhookx
RUN go mod download

COPY . .
RUN go install

FROM alpine:3.15

COPY --from=build-env /go/bin/webhookx /usr/local/bin

RUN apk add --no-cache gcompat

EXPOSE 8080

CMD ["webhookx", "start"]
