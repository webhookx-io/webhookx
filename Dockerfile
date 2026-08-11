FROM node:24-alpine AS ui

WORKDIR /webhookx

COPY ui/package.json ui/package-lock.json ./ui/
RUN npm --prefix ui ci

COPY openapi.yml ./openapi.yml
COPY ui/ ./ui/
RUN npm --prefix ui run build

FROM golang:1.26.5 AS build-env

WORKDIR /go/src/webhookx-io/webhookx

COPY api/license api/license
COPY go.mod go.sum ./
RUN go mod download

COPY --exclude=ui/dist . .
COPY --from=ui /webhookx/ui/dist ./ui/dist
RUN make build

FROM alpine:3.22

COPY --from=build-env /go/src/webhookx-io/webhookx/webhookx /usr/local/bin

EXPOSE 9600
EXPOSE 9601
EXPOSE 9602
EXPOSE 9605


CMD ["webhookx", "start"]
