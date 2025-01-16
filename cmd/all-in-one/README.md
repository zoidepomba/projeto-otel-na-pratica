# Pacote `cmd/all-in-one`

Para subir o módulo all-in-one local, você precisa ter o NATS rodando localmente

1.1 Para rodar o NATS localmente, você pode usar o Docker com o seguinte comando:
$ docker run -d --name nats-server -p 4222:4222 nats:latest --jetstream

1.2 Após subir o NATS, você precisará configurá-lo. Para isso, é necessário ter o binário do nats-cli. E possível baixar o binário através do seguinte link: https://github.com/nats-io/natscli/releases/tag/v0.1.6. 

1.3 Após instalar o nats-cli, execute o seguinte comando para configurar o stream:

```terminal
$ nats -s localhost:4222 stream create payments --subjects "payment.process" --storage memory --replicas 1 --retention=limits --discard=old --max-msgs 1_000_000 --max-msgs-per-subject 100_000 --max-bytes 4GiB --max-age 1d --max-msg-size 10MiB --dupe-window 2m --allow-rollup --no-deny-delete --no-deny-purge 
 ```

Após todas as configurações, o all-in-one estará pronto para subir.

Comando para executar a aplicação

```terminal
$ go run ./cmd/all-in-one/
```

Exemplos de Curl

```terminal
$ curl localhost:8080/payments
$ curl -X POST localhost:8080/users -d '{"id": "jpkroehling"}'
$ curl -X POST localhost:8080/subscriptions -d '{"id": "jpkroehling", "user_id":"jpkroehling", "plan_id":"silver"}'
$ curl -X POST localhost:8080/payments -d '{"id": "some-uuid", "subscription_id":"jpkroehling", "amount":99, "status":"FAILED"}'
$ nats -s localhost:4222 stream view payments
```
