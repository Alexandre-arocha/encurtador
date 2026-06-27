
<<<<<<< HEAD
=======
Encurtador de URL com slug customizável, expiração e dashboard de cliques
(referrer, device, geo). Backend em Go (Gin + pgx/sqlc + River), front em
Next.js.

## Como rodar em dev

Suba o Postgres local:

```powershell
docker compose up -d postgres
```

Crie o arquivo de ambiente a partir do exemplo:

```powershell
Copy-Item .env.example .env
```

Suba o backend. As migrations aplicam no boot:

```powershell
cd backend
go run ./cmd/api
```

Em outro terminal, suba o front:

```powershell
cd web
npm run dev
```

Healthcheck do backend:

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

## Testes

Os testes padrao nao exigem banco:

```powershell
cd backend
go test ./...
```

Para incluir contratos HTTP e stats contra Postgres real, configure
`TEST_DATABASE_URL` apontando para um banco descartavel. Os testes limpam
`links`, `clicks` e `link_daily` nesse banco.

Valide também o front:

```powershell
cd web
npm run build
npm audit --audit-level=moderate
```

## Estrutura

Ver [`CLAUDE.md`](CLAUDE.md) para estrutura de pastas, modelo de dados, API e o
plano por fases. O [`codex-tasks.md`](codex-tasks.md) descreve a faixa de
trabalho isolada do Codex para enrich e agregações de stats.

## GeoIP

O enriquecimento geográfico usa o MaxMind GeoLite2-City (`.mmdb`). O arquivo não
é versionado nem baixado automaticamente; baixe manualmente e aponte
`GEOLITE_DB_PATH` para ele, por exemplo `./data/GeoLite2-City.mmdb`.

Sem o arquivo, o enrich segue funcionando best-effort: campos de geo ficam
vazios e o redirect/stats continuam funcionando.
>>>>>>> 7a8302d (1.3)
