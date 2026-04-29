# Backup And Restore Runbook

This runbook covers the current local Docker-backed backend stack: PostgreSQL, MinIO, Redis, and RabbitMQ.

## Scope

- PostgreSQL is the source of truth for users, branch data, finance records, files metadata, notifications, and outbox rows.
- MinIO stores uploaded file bytes. PostgreSQL stores only file metadata and object keys.
- Redis is a cache; it can be restored from empty state.
- RabbitMQ carries transient events; durable event recovery comes from PostgreSQL `outbox_events`.

## Backup

Create a timestamped folder:

```powershell
$stamp = Get-Date -Format "yyyyMMdd-HHmmss"
New-Item -ItemType Directory -Force "backups\$stamp" | Out-Null
```

Backup PostgreSQL:

```powershell
docker compose exec -T postgres pg_dump -U kingsway -d kingsway --format=custom --file=/tmp/kingsway.dump
docker compose cp postgres:/tmp/kingsway.dump "backups\$stamp\kingsway.dump"
```

Backup MinIO buckets:

```powershell
docker compose exec -T minio mc alias set local http://localhost:9000 kingsway kingsway-secret
docker compose exec -T minio mc mirror local/kingsway-standard /tmp/kingsway-standard
docker compose exec -T minio mc mirror local/kingsway-special /tmp/kingsway-special
docker compose cp minio:/tmp/kingsway-standard "backups\$stamp\kingsway-standard"
docker compose cp minio:/tmp/kingsway-special "backups\$stamp\kingsway-special"
```

Backup RabbitMQ definitions:

```powershell
docker compose exec -T rabbitmq rabbitmqctl export_definitions /tmp/rabbitmq-definitions.json
docker compose cp rabbitmq:/tmp/rabbitmq-definitions.json "backups\$stamp\rabbitmq-definitions.json"
```

Redis is currently cache-only. A Redis snapshot is optional:

```powershell
docker compose exec -T redis redis-cli SAVE
docker compose cp redis:/data/dump.rdb "backups\$stamp\redis-dump.rdb"
```

## Restore

Stop API workers before restoring to avoid writes during restore:

```powershell
# Stop any local api-server/go run process first, then:
docker compose up -d postgres minio rabbitmq redis
```

Restore PostgreSQL into an empty database:

```powershell
docker compose cp "backups\$stamp\kingsway.dump" postgres:/tmp/kingsway.dump
docker compose exec -T postgres dropdb -U kingsway --if-exists kingsway
docker compose exec -T postgres createdb -U kingsway kingsway
docker compose exec -T postgres pg_restore -U kingsway -d kingsway --clean --if-exists /tmp/kingsway.dump
```

Restore MinIO buckets:

```powershell
docker compose exec -T minio mc alias set local http://localhost:9000 kingsway kingsway-secret
docker compose exec -T minio mc mb --ignore-existing local/kingsway-standard
docker compose exec -T minio mc mb --ignore-existing local/kingsway-special
docker compose cp "backups\$stamp\kingsway-standard" minio:/tmp/kingsway-standard
docker compose cp "backups\$stamp\kingsway-special" minio:/tmp/kingsway-special
docker compose exec -T minio mc mirror --overwrite /tmp/kingsway-standard local/kingsway-standard
docker compose exec -T minio mc mirror --overwrite /tmp/kingsway-special local/kingsway-special
```

Restore RabbitMQ definitions if queues/exchanges were changed outside code:

```powershell
docker compose cp "backups\$stamp\rabbitmq-definitions.json" rabbitmq:/tmp/rabbitmq-definitions.json
docker compose exec -T rabbitmq rabbitmqctl import_definitions /tmp/rabbitmq-definitions.json
```

Redis can be left empty. Salary swap cache keys and other cache values will be rebuilt by normal API usage.

## Verification

Run these checks after restore:

```powershell
docker compose ps
go test -c ./internal/integration -o tmp\integration.test.exe
$env:KINGSWAY_INTEGRATION='1'; Push-Location internal\integration; ..\..\tmp\integration.test.exe; Pop-Location
go build -o tmp\api-server.exe ./cmd/api-server
```

Then start the API and verify:

- `GET /healthz` returns `{"status":"ok"}`.
- `GET /readyz` returns status `ok`.
- `GET /v1/admin/outbox?status=failed` as Owner shows failed outbox rows, if any.
- Existing uploaded files can produce `GET /v1/files/{file_id}/download-url`.
