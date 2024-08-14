# WebhookX [![Build Status](https://github.com/webhookx-io/webhookx/actions/workflows/test.yml/badge.svg)](https://github.com/webhookx-io/webhookx/actions/workflows/test.yml) [![Build Status](https://github.com/webhookx-io/webhookx/actions/workflows/lint.yml/badge.svg)](https://github.com/webhookx-io/webhookx/actions/workflows/lint.yml)

[![Join Slack](https://img.shields.io/badge/Slack-4285F4?logo=slack&logoColor=white)](https://join.slack.com/t/webhookx/shared_invite/zt-2o4b6hv45-mWm6_WUcQP9qEf1nOxhrrg)
[![Follow on Twitter](https://img.shields.io/badge/twitter-1DA1F2?logo=twitter&logoColor=white)](https://twitter.com/webhookx)

WebhookX is a webhooks gateway.

## Features

## Roadmap

- [x] Workspace Isolation
- [ ] Ingest/Proxy: a module that expose an http listener to receive events.
- [ ] Observability: distributed tracing

## Todo list

- [ ] manual retry an attempt
- [ ] HTTPS port
- [ ] Middleware
- [ ] OpenAPI

## Installation

```shell
$ docker build . -t webhookx-io/webhookx:latest
$ docker compose up
```

```shell
$ curl http://localhost:8080
```

## Contributing

## License
