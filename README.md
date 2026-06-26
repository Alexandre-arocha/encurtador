# Encurtador com analytics

Encurtador de URL com slug customizável, expiração e dashboard de cliques
(referrer, device, geo). Backend em Go (Gin + pgx/sqlc + River), front em
Next.js.






<<<<<<< HEAD
=======
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

## Smoke local

Com o Postgres e o backend rodando, valide o fluxo criar link -> redirect ->
stats:

```powershell
.\scripts\smoke-local.ps1
```

O script usa por padrão `http://localhost:8080` e a API key de `.env.example`.
Para sobrescrever:

```powershell
.\scripts\smoke-local.ps1 -ApiBaseUrl http://localhost:8080 -ApiKey sua-chave
```

## Testes de integracao

Os testes padrao nao exigem banco:

```powershell
cd backend
go test ./...
```

Para incluir contratos HTTP e stats contra Postgres real, configure
`TEST_DATABASE_URL` apontando para um banco descartavel. Os testes limpam
`links`, `clicks` e `link_daily` nesse banco.

## Estrutura

Ver [`CLAUDE.md`](CLAUDE.md) para estrutura de pastas, modelo de dados, API e
o plano por fases.

## GeoIP

O enriquecimento geográfico usa o **MaxMind GeoLite2-City** (`.mmdb`). O arquivo
**não** é versionado nem baixado automaticamente — baixe manualmente e aponte
`GEOLITE_DB_PATH` para ele (padrão: `./data/GeoLite2-City.mmdb`). Sem o arquivo,
o enrich segue funcionando best-effort: campos de geo ficam vazios.
>>>>>>> 590f707 (1.2)
