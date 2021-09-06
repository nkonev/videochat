module nkonev.name/chat

require (
	contrib.go.opencensus.io/exporter/jaeger v0.2.0
	github.com/araddon/dateparse v0.0.0-20200409225146-d820a6159ab1
	github.com/centrifugal/centrifuge v0.18.4
	github.com/centrifugal/protocol v0.7.3
	github.com/getlantern/deepcopy v0.0.0-20160317154340-7f45deb8130a
	github.com/go-ozzo/ozzo-validation/v4 v4.2.1
	github.com/golang-migrate/migrate/v4 v4.11.0
	github.com/gomodule/redigo v1.8.5
	github.com/google/uuid v1.3.0
	github.com/gorilla/websocket v1.4.2
	github.com/grokify/html-strip-tags-go v0.0.1
	github.com/guregu/null v4.0.0+incompatible
	github.com/isayme/go-amqp-reconnect v0.0.0-20210303120416-fc811b0bcda2
	github.com/jackc/pgx/v4 v4.8.1
	github.com/labstack/echo/v4 v4.1.16
	github.com/m7shapan/njson v1.0.3
	github.com/microcosm-cc/bluemonday v1.0.3
	github.com/nkonev/jaeger-uber-propagation-compat v0.0.0-20200708125206-e763f0a72519
	github.com/oliveagle/jsonpath v0.0.0-20180606110733-2e52cf6e6852
	github.com/sirupsen/logrus v1.6.0
	github.com/spf13/viper v1.7.0
	github.com/streadway/amqp v1.0.0
	github.com/stretchr/testify v1.7.0
	go.opencensus.io v0.22.4
	go.uber.org/fx v1.12.0
)

go 1.16
