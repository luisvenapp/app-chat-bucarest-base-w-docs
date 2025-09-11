# Docker Compose — Chat Messages API

## Requisitos
- Docker Desktop / Docker Compose v2
- (Opcional) Agente SSH configurado si usas repos privados para el build

## Archivos relevantes
- `docker-compose.yml`
- `.env` (variables de entorno)
- `migrations/postgres/0001_init.sql` (migraciones automáticas de Postgres)
- `migrations/cassandra/0001_init.cql` (migraciones de Scylla/Cassandra)
- `dockerfile` (build multi-stage)
- `config/config.json` (valores por defecto de la app)

## Variables de entorno (.env)
Se provee un `.env` de ejemplo en la raíz. Puedes modificarlo para DEV/PROD.

Ejecuta:
```
docker compose --env-file .env up -d --build
```

## Servicios
- `postgres` — DB relacional (migraciones SQL automáticas al primer arranque del volumen)
- `scylla` — Base NoSQL (compatible Cassandra)
- `scylla-init` — Job que aplica `0001_init.cql` contra Scylla cuando el servicio está healthy
- `redis` — Cache y KV store
- `nats` — Mensajería + JetStream
- `app` — API gRPC/Connect (puerto 8080)

## Ciclo de vida
1. `postgres` levanta y aplica `migrations/postgres/0001_init.sql`
2. `scylla` levanta y se marca healthy
3. `scylla-init` aplica `migrations/cassandra/0001_init.cql` contra `scylla`
4. `redis` y `nats` quedan healthy
5. `app` se construye y arranca en `:8080`

## Comandos útiles
- Ver logs de la app:
```
docker compose logs -f app
```
- Reconstruir:
```
docker compose build --no-cache app
```
- Bajar todo:
```
docker compose down -v
```

## Notas
- Para producción, cambia `USE_SCYLLADB` según tu despliegue.
- Ajusta `chat.key` y `chat.iv` a valores seguros.
- Si usas módulos privados en el build, asegúrate de tener configurado el SSH agent.
