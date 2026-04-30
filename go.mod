module github.com/webhookx-io/webhookx

go 1.26.2

replace github.com/webhookx-io/webhookx/api/license => ./api/license

require (
	github.com/Masterminds/squirrel v1.5.4
	github.com/asaskevich/EventBus v0.0.0-20200907212545-49d423059eef
	github.com/aws/aws-sdk-go-v2 v1.41.7
	github.com/aws/aws-sdk-go-v2/config v1.32.17
	github.com/aws/aws-sdk-go-v2/credentials v1.19.16
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.41.7
	github.com/creasty/defaults v1.8.0
	github.com/dop251/goja v0.0.0-20260311135729-065cd970411c
	github.com/elazarl/goproxy v1.8.3
	github.com/getkin/kin-openapi v0.137.0
	github.com/go-kit/kit v0.13.0
	github.com/go-playground/form v3.1.4+incompatible
	github.com/go-playground/validator/v10 v10.30.2
	github.com/go-redis/redis_rate/v10 v10.0.1
	github.com/go-redsync/redsync/v4 v4.16.0
	github.com/go-resty/resty/v2 v2.17.2
	github.com/golang-migrate/migrate/v4 v4.19.1
	github.com/gorilla/mux v1.8.1
	github.com/hashicorp/golang-lru/v2 v2.0.7
	github.com/hashicorp/vault/api v1.23.0
	github.com/hashicorp/vault/api/auth/approle v0.12.0
	github.com/hashicorp/vault/api/auth/kubernetes v0.12.0
	github.com/jackc/pgerrcode v0.0.0-20250907135507-afb5586c32a6
	github.com/jackc/pgx/v5 v5.9.2
	github.com/jmoiron/sqlx v1.4.0
	github.com/lib/pq v1.12.3
	github.com/mattn/go-sqlite3 v1.14.44
	github.com/mitchellh/mapstructure v1.5.0
	github.com/onsi/ginkgo/v2 v2.28.3
	github.com/onsi/gomega v1.40.0
	github.com/pkg/errors v0.9.1
	github.com/redis/go-redis/v9 v9.19.0
	github.com/robfig/cron/v3 v3.0.1
	github.com/rs/zerolog v1.35.1
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.2
	github.com/satori/go.uuid v1.2.0
	github.com/segmentio/ksuid v1.0.4
	github.com/spf13/cobra v1.10.2
	github.com/stretchr/testify v1.11.1
	github.com/stripe/stripe-go/v84 v84.4.1
	github.com/tetratelabs/wazero v1.11.0
	github.com/tidwall/gjson v1.18.0
	github.com/vmihailenco/msgpack/v5 v5.4.1
	github.com/webhookx-io/webhookx/api/license v0.1.0
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.68.0
	go.opentelemetry.io/contrib/propagators/autoprop v0.68.0
	go.opentelemetry.io/otel v1.43.0
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v1.43.0
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v1.43.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.43.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.43.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.43.0
	go.opentelemetry.io/otel/metric v1.43.0
	go.opentelemetry.io/otel/sdk v1.43.0
	go.opentelemetry.io/otel/sdk/metric v1.43.0
	go.opentelemetry.io/otel/trace v1.43.0
	go.uber.org/mock v0.6.0
	go.uber.org/zap v1.28.0
	golang.org/x/sync v0.20.0
	google.golang.org/grpc v1.80.0
	gopkg.in/yaml.v3 v3.0.1
)

require github.com/felixge/httpsnoop v1.0.4 // indirect

require (
	github.com/Masterminds/semver/v3 v3.4.0 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.23 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.23 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.23 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.23 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.0.11 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.30.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.35.21 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.42.1 // indirect
	github.com/aws/smithy-go v1.25.1 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/dlclark/regexp2 v1.12.0 // indirect
	github.com/go-jose/go-jose/v4 v4.1.4 // indirect
	github.com/go-openapi/jsonpointer v0.21.0 // indirect
	github.com/go-openapi/swag v0.23.0 // indirect
	github.com/go-sourcemap/sourcemap v2.1.4+incompatible // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.8 // indirect
	github.com/hashicorp/go-rootcerts v1.0.2 // indirect
	github.com/hashicorp/go-secure-stdlib/parseutil v0.2.0 // indirect
	github.com/hashicorp/go-secure-stdlib/strutil v0.1.2 // indirect
	github.com/hashicorp/go-sockaddr v1.0.7 // indirect
	github.com/hashicorp/hcl v1.0.1-vault-7 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.22 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826 // indirect
	github.com/oasdiff/yaml v0.0.9 // indirect
	github.com/oasdiff/yaml3 v0.0.12 // indirect
	github.com/perimeterx/marshmallow v1.1.5 // indirect
	github.com/ryanuber/go-glob v1.0.0 // indirect
	github.com/tidwall/match v1.2.0 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	github.com/woodsbury/decimal128 v1.3.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/propagators/aws v1.43.0 // indirect
	go.opentelemetry.io/contrib/propagators/b3 v1.43.0 // indirect
	go.opentelemetry.io/contrib/propagators/jaeger v1.43.0 // indirect
	go.opentelemetry.io/contrib/propagators/ot v1.43.0 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/mod v0.35.0 // indirect
	golang.org/x/time v0.15.0 // indirect
)

require (
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.24 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/gabriel-vasile/mimetype v1.4.13 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/pprof v0.0.0-20260402051712-545e8a4df936 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.29.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/lann/builder v0.0.0-20180802200727-47ae307949d0 // indirect
	github.com/lann/ps v0.0.0-20150810152359-62de8c46ede0 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	go.opentelemetry.io/proto/otlp v1.10.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/crypto v0.50.0 // indirect
	golang.org/x/net v0.53.0 // indirect
	golang.org/x/sys v0.43.0 // indirect
	golang.org/x/text v0.36.0 // indirect
	golang.org/x/tools v0.44.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260427160629-7cedc36a6bc4 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260427160629-7cedc36a6bc4 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/go-playground/assert.v1 v1.2.1 // indirect
)
