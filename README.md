# WebhookX [![Go Report Card](https://goreportcard.com/badge/github.com/webhookx-io/webhookx)](https://goreportcard.com/report/github.com/webhookx-io/webhookx) [![Build Status](https://github.com/webhookx-io/webhookx/actions/workflows/test.yml/badge.svg)](https://github.com/webhookx-io/webhookx/actions/workflows/test.yml) [![Build Status](https://github.com/webhookx-io/webhookx/actions/workflows/lint.yml/badge.svg)](https://github.com/webhookx-io/webhookx/actions/workflows/lint.yml)

[![Join Slack](https://img.shields.io/badge/Slack-4285F4?logo=slack&logoColor=white)](https://join.slack.com/t/webhookx/shared_invite/zt-2o4b6hv45-mWm6_WUcQP9qEf1nOxhrrg)
[![Follow on Twitter](https://img.shields.io/badge/twitter-1DA1F2?logo=twitter&logoColor=white)](https://twitter.com/webhookx)

WebhookX is an open-source webhooks gateway for message receiving, processing, and delivering.


## Features

- **Admin API:** The admin API(:8080) provides a RESTful API for webhooks entities management.
- **Retries:** WebhookX automatically retries unsuccessful deliveries at configurable delays.
- **Fan out:** Events can be fan out to multiple destinations.
- **Declarative configuration:**  Managing your configuration through declarative configuration file, and be DevOps compliant.

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


## Getting started

##### 1. Create an endpoint

```
curl -X POST 'http://localhost:8080/workspaces/default/endpoints' \
--header 'Content-Type: application/json' \
--data '{
    "request": {
        "url": "https://httpbin.org/anything",
        "method": "POST"
    },
    "events": [
        "charge.succeeded"
    ]
}'
```

##### 2. Create a source

```
curl -X POST 'http://localhost:8080/workspaces/default/sources' \
--header 'accept: application/json' \
--header 'Content-Type: application/json' \
--data '{
  "path": "/",
  "methods": ["POST"]
}'
```

#### 3. Send an event to proxy

```
curl -X POST 'http://localhost:8081/' \
--header 'Content-Type: application/json' \
--data '{
    "event_type": "charge.succeeded",
    "data": {
        "key": "value"
    }
}'
```

#### 4. Retrieve delivery attemp

```
curl 'http://localhost:8080/workspaces/default/attempts'

{
    "total": 1,
    "data": [
        {
            "id": "c11b4011-623c-4aa0-8c54-4e4672799e15",
            "event_id": "dfc3fa33-4e0e-4b43-96ab-c1ad92de67f4",
            "endpoint_id": "97d5fecd-f912-43fb-9a22-2073931acbeb",
            "status": "SUCCESSFUL",
            "attempt_number": 1,
            "attempt_at": 1724336398,
            "request": {
                "method": "POST",
                "url": "https://httpbin.org/anything",
                "header": {},
                "body": "{\"key\": \"value\"}"
            },
            "response": {
                "status": 200,
                "header": {},
                "body": "{\n  \"args\": {}, \n  \"data\": \"{\\\"key\\\": \\\"value\\\"}\", \n  \"files\": {}, \n  \"form\": {}, \n  \"headers\": {\n    \"Accept-Encoding\": \"gzip\", \n    \"Content-Length\": \"16\", \n    \"Content-Type\": \"application/json; charset=utf-8\", \n    \"Host\": \"httpbin.org\", \n    \"User-Agent\": \"WebhookX/dev\", \n    \"X-Amzn-Trace-Id\": \"Root=1-66c7490f-044864fb41bb67160fe2a4e3\"\n  }, \n  \"json\": {\n    \"key\": \"value\"\n  }, \n  \"method\": \"POST\", \n  \"origin\": \"117.186.1.161\", \n  \"url\": \"https://httpbin.org/anything\"\n}\n"
            },
            "created_at": 1724307597,
            "updated_at": 1724307597
        }
    ]
}
```

## Runtime dependencies

The gateway requires the following runtime dependencies to work:

- PostgreSQL(>=13)
- Redis(>=4)

## Sponsoring

## Contributing

We ❤️ pull requests

## License
