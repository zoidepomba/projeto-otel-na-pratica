# Pacote `cmd`

A ser documentado.

Para visualização da instrumentação via stack do grafana favor usar o seguinte comando
```terminal
$docker run -p 3000:3000 -p 5317:5317 --rm -ti docker.io/grafana/otel-lgtm:latest
```

OTEL_SERVICE_NAME=subscriptions OTEL_EXPORTER_OTLP_INSECURE=true go run cmd/all-in-one/main.goOTEL_SERVICE_NAME=subscriptions OTEL_EXPORTER_OTLP_INSECURE=true go run cmd/all-in-one/main.go

Para rodar a aplicação local existe algumas dependencias, 