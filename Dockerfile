FROM golang:1.22 AS build-env

WORKDIR /go/src/webhookx-io/webhookx

COPY go.mod /go/src/webhookx-io/webhookx
COPY go.sum /go/src/webhookx-io/webhookx
RUN go mod download

COPY . .
RUN make build

FROM alpine:3.15

COPY --from=build-env /go/src/webhookx-io/webhookx/webhookx /usr/local/bin

EXPOSE 8080

CMD ["webhookx", "start"]
