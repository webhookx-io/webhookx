FROM golang:1.25.3 AS build-env

WORKDIR /go/src/webhookx-io/webhookx

COPY go.mod /go/src/webhookx-io/webhookx
COPY go.sum /go/src/webhookx-io/webhookx
RUN go mod download

COPY . .
RUN make build

FROM alpine:3.15

COPY --from=build-env /go/src/webhookx-io/webhookx/webhookx /usr/local/bin

EXPOSE 9600
EXPOSE 9601
EXPOSE 9602


CMD ["webhookx", "start"]
