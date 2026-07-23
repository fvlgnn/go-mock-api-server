# go-mock-api-server

Server HTTP minimale scritto in Go per creare API mock configurabili tramite
file JSON. È pensato per sviluppo, demo e test di integrazione, senza dipendenze
Go esterne e con un'immagine container `scratch` non-root.

## Funzionalità

- route configurabili tramite file JSON;
- metodi e path esatti, con risposta `405 Method Not Allowed` e header `Allow`;
- status code e response header personalizzabili;
- validazione completa della configurazione prima dell'avvio;
- più metodi consentiti sullo stesso path;
- configurazione runtime tramite variabili d'ambiente;
- directory dei mock montabile come volume read-only;
- endpoint `GET /-/health` e healthcheck integrato nell'immagine;
- timeout HTTP e arresto graceful su `SIGTERM`/`SIGINT`;
- log strutturati;
- immagini GHCR versionate per `linux/amd64` e `linux/arm64`.

## Avvio rapido con Docker

Le immagini vengono pubblicate in seguito alla creazione di un tag SemVer:

```sh
docker run --rm -p 8080:8080 \
  ghcr.io/fvlgnn/go-mock-api-server:latest
```

Verifica il server:

```sh
curl http://localhost:8080/-/health
curl http://localhost:8080/v1/users
```

Per ambienti riproducibili usa una versione esplicita, ad esempio:

```sh
docker pull ghcr.io/fvlgnn/go-mock-api-server:1.0.0
```

Il tag `latest` segue l'ultima release e non è consigliato in produzione o in
pipeline che richiedono build deterministiche.

## Configurazione delle route

Il server legge tutti i file con estensione `.json` presenti in `CONFIG_DIR`.
Ogni file descrive una singola coppia metodo/path:

```json
{
  "request": {
    "method": "GET",
    "path": "/v1/user"
  },
  "response": {
    "status": 200,
    "headers": {
      "X-Mock-Server": "go-mock-api-server"
    },
    "body": [
      { "id": 1, "name": "Tom" }
    ]
  }
}
```

Campi:

| Campo | Obbligatorio | Default | Descrizione |
|---|---:|---|---|
| `request.method` | sì | — | Metodo HTTP; viene normalizzato in maiuscolo |
| `request.path` | sì | — | Path assoluto, pulito ed esatto |
| `response.status` | no | `200` | Status HTTP tra 200 e 599 |
| `response.headers` | no | `{}` | Header aggiunti alla risposta |
| `response.body` | sì | — | Qualsiasi valore JSON, incluso `null` |

Se `Content-Type` non è configurato, il server usa
`application/json; charset=utf-8`.

### Regole di validazione

Il processo termina prima di aprire la porta se:

- la directory non esiste, non è leggibile o non contiene definizioni JSON;
- un file contiene JSON non valido, campi sconosciuti o più valori JSON;
- metodo, path, status o header non sono validi;
- manca `response.body`;
- due file definiscono la stessa coppia metodo e path;
- una route tenta di usare `/-/health`, riservato al server.

È consentito definire metodi diversi sullo stesso path, per esempio `GET /users`
e `POST /users`. Le route sono esatte: `/users` non intercetta `/users/1`.

## Configurazione runtime

| Variabile | Default locale | Default container | Descrizione |
|---|---|---|---|
| `CONFIG_DIR` | `config` | `/config` | Directory delle definizioni JSON |
| `SERVER_PORT` | `8080` | `8080` | Porta TCP, da 1 a 65535 |
| `READ_TIMEOUT` | `5s` | `5s` | Timeout lettura richiesta |
| `WRITE_TIMEOUT` | `10s` | `10s` | Timeout scrittura risposta |
| `IDLE_TIMEOUT` | `1m` | `1m` | Timeout connessioni keep-alive inattive |
| `SHUTDOWN_TIMEOUT` | `10s` | `10s` | Tempo massimo per lo shutdown graceful |

I timeout usano il formato `time.Duration` di Go, ad esempio `500ms`, `10s` o
`2m`.

Esempio:

```sh
SERVER_PORT=9090 CONFIG_DIR=./my-mocks go run .
```

## Usare un volume di configurazione

Il modo consigliato per usare mock personalizzati non richiede una nuova build:

```sh
docker run --rm \
  -p 8080:8080 \
  -v "$(pwd)/my-mocks:/config:ro" \
  ghcr.io/fvlgnn/go-mock-api-server:1.0.0
```

Il mount read-only evita che il container possa modificare i file host. La
configurazione viene caricata una volta all'avvio; dopo una modifica ai JSON è
necessario ricreare o riavviare il container.

Con Docker Compose:

```yaml
services:
  app-be:
    image: ghcr.io/fvlgnn/go-mock-api-server:1.0.0
    environment:
      SERVER_PORT: "8080"
      CONFIG_DIR: /config
    volumes:
      - ./app-be:/config:ro
    ports:
      - "8080:8080"
    security_opt:
      - no-new-privileges:true
```

Questa è la modalità prevista per
[`fvlgnn/caddy-reverse-proxy-for-container`](https://github.com/fvlgnn/caddy-reverse-proxy-for-container).

## Health check

```sh
curl --fail http://localhost:8080/-/health
```

Risposta:

```json
{"status":"ok","version":"v1.0.0"}
```

L'immagine definisce anche un `HEALTHCHECK` senza aggiungere shell o utility:
richiama lo stesso binario con il comando `healthcheck`.

## Esecuzione locale

Richiede Go 1.26 o successivo:

```sh
go run .
```

Comandi disponibili:

```sh
go run . version
go run . healthcheck
```

Il secondo comando verifica un server già in ascolto su `SERVER_PORT`.

## Build e test

```sh
make check
make build
make docker-build
```

Equivalenti diretti:

```sh
gofmt -w .
go vet ./...
go test -race -cover ./...
docker build -t go-mock-api-server:local .
```

## CI, release e immagini GHCR

La workflow `CI` viene eseguita su pull request e push su `main`. Controlla
formattazione, `go vet`, test con race detector, build Go e smoke test
dell'immagine container.

La workflow `Release` parte esclusivamente da tag SemVer nel formato
`vMAJOR.MINOR.PATCH`. Prima di creare il tag, sposta le modifiche dalla sezione
`[Unreleased]` di [CHANGELOG.md](CHANGELOG.md) a una nuova sezione con la stessa
versione del tag:

```markdown
## [1.1.0] - 2026-10-15

### Added

- Descrizione della nuova funzionalità.
```

Poi crea e pubblica il tag:

```sh
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

Dopo il superamento dei test, la workflow:

1. pubblica un manifest multi-arch per `linux/amd64` e `linux/arm64`;
2. crea i tag immagine `1.0.0`, `1.0`, `1` e `latest`;
3. aggiunge label OCI, SBOM e provenance;
4. estrae da `CHANGELOG.md` soltanto la sezione corrispondente al tag;
5. crea la GitHub Release usando quella sezione come descrizione.

La pubblicazione fallisce se la sezione del changelog non esiste o è vuota:
in questo modo una release non può essere creata con note mancanti o relative
a una versione diversa.

L'immagine risultante è:

```text
ghcr.io/fvlgnn/go-mock-api-server:<version>
```

La repository deve permettere a GitHub Actions di scrivere nei package. La
workflow usa il `GITHUB_TOKEN` e non richiede PAT o secret aggiuntivi.

Dopo la prima pubblicazione, verifica nelle impostazioni del package GHCR che
la visibilità sia **Public**: la visibilità dei package è distinta da quella
del repository. Verifica inoltre che il package sia collegato a questo
repository, così eredita correttamente i permessi della workflow.

## Aggiornamenti delle dipendenze

Dependabot controlla trimestralmente toolchain Go, immagine Docker e GitHub
Actions. Per un progetto piccolo questa cadenza riduce il rumore rispetto a un
controllo mensile, senza accumulare un anno di aggiornamenti e possibili
incompatibilità in una sola volta.

La cadenza riguarda gli aggiornamenti ordinari di versione. Gli eventuali
security update di Dependabot sono gestiti separatamente da GitHub e possono
essere aperti appena viene rilevata una dipendenza vulnerabile, se la relativa
funzionalità è abilitata nelle impostazioni del repository.

## Sicurezza e limiti

Il container finale:

- usa `scratch`;
- contiene solo il binario statico e i mock predefiniti;
- gira come UID/GID non-root `65532:65532`;
- non include shell o package manager.

Il server non implementa autenticazione, TLS, rate limiting o CORS. Sono
responsabilità del reverse proxy o dell'ambiente che lo espone. Non pubblicare
mock o dati sensibili su reti non fidate. Consulta [SECURITY.md](SECURITY.md).

## License

Distribuito con licenza [MIT](LICENSE).
