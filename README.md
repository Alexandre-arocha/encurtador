# Encurtador com analytics

Encurtador de URL com slug customizável, expiração e dashboard de cliques
(referrer, device, geo). Backend em Go (Gin + pgx/sqlc + River), front em
Next.js.

> ⚠️ **As regras do projeto estão em [`CLAUDE.md`](CLAUDE.md).** A mais
> importante: o agente de IA **nunca** faz commit — todo commit é manual.

## Pré-requisitos

- Go 1.23+
- Docker (para o Postgres local)
- Node 18+ (apenas para o front, Fase 5)

## Rodando o backend (dev)

```powershell
# 1. Subir o Postgres
docker compose up -d postgres

# 2. Criar o .env a partir do exemplo
Copy-Item .env.example .env

# 3. Subir o backend (migrations aplicam no boot)
cd backend
go run ./cmd/api
```

Healthcheck:

```powershell
curl http://localhost:8080/healthz
# {"status":"ok","db":"up"}
```

## Estrutura



O enriquecimento geográfico usa o **MaxMind GeoLite2-City** (`.mmdb`). O arquivo
**não** é versionado nem baixado automaticamente — baixe manualmente e aponte
`GEOLITE_DB_PATH` para ele (padrão: `./data/GeoLite2-City.mmdb`). Sem o arquivo,
o enrich segue funcionando best-effort: campos de geo ficam vazios.
