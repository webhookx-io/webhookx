# WebhookX [![Build Status](https://github.com/webhookx-io/webhookx/actions/workflows/test.yml/badge.svg)](https://github.com/webhookx-io/webhookx/actions/workflows/test.yml) [![Build Status](https://github.com/webhookx-io/webhookx/actions/workflows/lint.yml/badge.svg)](https://github.com/webhookx-io/webhookx/actions/workflows/lint.yml)

[![Join Slack](https://img.shields.io/badge/Slack-4285F4?logo=slack&logoColor=white)](https://join.slack.com/t/webhookx/shared_invite/zt-2o4b6hv45-mWm6_WUcQP9qEf1nOxhrrg)
[![Follow on Twitter](https://img.shields.io/badge/twitter-1DA1F2?logo=twitter&logoColor=white)](https://twitter.com/webhookx)

WebhookX is an open-source webhooks gateway for message receiving, processing, and delivering.


## Features

- **Admin API:** The admin API(:8080) provides a RESTful API for webhooks entities management.
- **Retries:** WebhookX automatically retries unsuccessful deliveries at configurable delays.
- **Fan out:** Events can be fan out to multiple destinations.


## Roadmap

- [x] Workspace 
- [ ] Data retention policy
- [x] OpenAPI
- [ ] Insight admin APIs
- [ ] Observability(o11y) including tracing and metrics
- [ ] Declarative configuration management

### Outbound

- [ ] Authentication
- [ ] Manually retry

### Inbound

- [x] Inbound Gateway
- [ ] Middlewares/Plugins
- [ ] Authentication
- [ ] Event Transformer

## Installation

```shell
$ docker build . -t webhookx-io/webhookx:latest
$ docker compose up
```

```shell
$ curl http://localhost:8080
```

## Contributing

We ❤️ pull requests

## License
