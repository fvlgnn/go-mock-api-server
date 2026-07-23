# Note del maintainer

Questo file è versionato come promemoria operativo del maintainer, ma non fa parte della documentazione utente. Non inserire password, token, dati personali o configurazioni riservate.

## CHANGELOG: come aggiornarlo

Durante lo sviluppo, aggiungere le modifiche sotto `## [Unreleased]`, usando le
categorie appropriate:

```markdown
## [Unreleased]

### Added

- Nuova funzionalità.

### Changed

- Comportamento modificato.

### Fixed

- Bug corretto.

## [1.0.0] - 2026-07-23
```

Quando si prepara una release, trasformare il contenuto Unreleased in una nuova
sezione versionata. La nuova versione deve essere inserita immediatamente sotto
`## [Unreleased]` e sopra la release precedente:

```markdown
## [Unreleased]

## [1.1.0] - 2026-10-15

### Added

- Nuova funzionalità.

### Fixed

- Bug corretto.

## [1.0.0] - 2026-07-23
```

Quindi:

1. `Unreleased` rimane sempre come prima sezione ed è nuovamente vuota.
2. La release più recente viene subito dopo `Unreleased`.
3. Le release precedenti rimangono sotto, dalla più recente alla più vecchia.
4. La versione nel titolo non contiene `v`: usare `[1.1.0]`.
5. Il tag Git contiene `v`: usare `v1.1.0`.
6. La versione della sezione deve corrispondere esattamente al tag senza `v`.

Aggiornare anche i link in fondo al changelog:

```markdown
[Unreleased]: https://github.com/fvlgnn/go-mock-api-server/compare/v1.1.0...HEAD
[1.1.0]: https://github.com/fvlgnn/go-mock-api-server/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/fvlgnn/go-mock-api-server/releases/tag/v1.0.0
```

Lo script di release legge soltanto il contenuto compreso tra:

```text
## [VERSIONE-CORRENTE]
```

e la successiva intestazione:

```text
## [VERSIONE-PRECEDENTE]
```

La sezione `Unreleased` non viene pubblicata nella descrizione della release.

## Sviluppo locale

Avviare il server con i mock predefiniti:

```sh
go run .
```

Avviare con porta e directory personalizzate:

```sh
SERVER_PORT=9090 CONFIG_DIR=./my-mocks go run .
```

Controllare versione del binario:

```sh
go run . version
```

Verificare un server già avviato:

```sh
go run . healthcheck
curl --fail http://localhost:8080/-/health
```

Provare gli endpoint inclusi:

```sh
curl http://localhost:8080/v1/users
curl http://localhost:8080/v1/user/4
curl -X POST http://localhost:8080/v1/user/create
curl -X DELETE http://localhost:8080/v1/user/delete/3
```

## Test e controlli

Eseguire tutti i controlli:

```sh
make check
```

Comandi separati:

```sh
gofmt -w .
go vet ./...
go test -race -cover ./...
git diff --check
```

Build del binario:

```sh
make build
./bin/go-mock-api-server version
```

## Ambiente container locale

Costruire l'immagine:

```sh
make docker-build
```

Oppure:

```sh
docker build \
  --build-arg VERSION=dev \
  --build-arg COMMIT="$(git rev-parse --short HEAD)" \
  --build-arg BUILD_DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -t go-mock-api-server:local .
```

Avviare con le configurazioni incorporate:

```sh
docker run --rm \
  --name go-mock-api-server \
  -p 8080:8080 \
  go-mock-api-server:local
```

Avviare in background con un volume read-only:

```sh
docker run -d \
  --name go-mock-api-server \
  -p 8080:8080 \
  -v "$(pwd)/config:/config:ro" \
  go-mock-api-server:local
```

Controllare stato, health e log:

```sh
docker inspect \
  --format='{{.State.Health.Status}}' \
  go-mock-api-server

curl --fail http://localhost:8080/-/health
docker logs -f go-mock-api-server
```

Arrestare e rimuovere il container:

```sh
docker rm -f go-mock-api-server
```

## Preparazione di una release

Scegliere la versione secondo SemVer:

- patch `1.0.1`: correzioni retrocompatibili;
- minor `1.1.0`: nuove funzionalità retrocompatibili;
- major `2.0.0`: modifiche incompatibili.

Checklist:

1. Spostare le note da `[Unreleased]` alla nuova sezione versionata.
2. Inserire la nuova sezione sopra la release precedente.
3. Aggiornare i link in fondo a `CHANGELOG.md`.
4. Verificare che data e versione siano corrette.
5. Eseguire test e validazione delle release notes.

Esempio per `v1.1.0`:

```sh
make check
sh .github/scripts/release-notes.sh v1.1.0 /tmp/release-notes.md
cat /tmp/release-notes.md
```

Controllare le modifiche:

```sh
git status
git diff --check
git diff
```

Commit e push:

```sh
git add .
git commit -m "prepare release v1.1.0"
git push origin main
```

Creare un tag annotato sul commit appena pubblicato:

```sh
git tag -a v1.1.0 -m "Release v1.1.0"
git push origin v1.1.0
```

Il push del tag avvia la workflow Release, che:

1. esegue test e vet;
2. pubblica l'immagine GHCR per amd64 e arm64;
3. crea i tag immagine `1.1.0`, `1.1`, `1` e `latest`;
4. estrae la sezione `[1.1.0]` da `CHANGELOG.md`;
5. usa quella sezione come descrizione della GitHub Release.

## Avvio manuale di una release esistente

Se il tag esiste ma la workflow non è partita, per esempio perché GitHub
Actions era disabilitato, non eliminare e non spostare il tag. Avvia la release
manualmente:

```sh
gh workflow run release.yml -f release_tag=v1.1.0
gh run watch
```

Oppure usa GitHub:

1. aprire `Actions`;
2. selezionare `Release`;
3. selezionare `Run workflow`;
4. inserire il tag esistente, ad esempio `v1.1.0`;
5. avviare la workflow.

La workflow esegue il checkout del tag indicato e usa commit, sorgenti e
CHANGELOG presenti in quel tag. Non pubblica il contenuto corrente di `main`.

Controllare la workflow:

```sh
gh run list --workflow Release
gh run watch
```

Controllare la release:

```sh
gh release view v1.1.0
```

Verificare l'immagine pubblicata:

```sh
docker pull ghcr.io/fvlgnn/go-mock-api-server:1.1.0
docker run --rm ghcr.io/fvlgnn/go-mock-api-server:1.1.0 version
```

## Se la release fallisce

Verificare nell'ordine:

```sh
sh .github/scripts/release-notes.sh v1.1.0 /tmp/release-notes.md
cat /tmp/release-notes.md
git show v1.1.0:CHANGELOG.md
gh run list --workflow Release
```

Cause comuni:

- manca `## [1.1.0]` nel changelog presente nel commit taggato;
- sezione `[1.1.0]` vuota;
- tag diverso dal formato `vMAJOR.MINOR.PATCH`;
- tag creato sul commit sbagliato;
- Actions non ha permesso `packages: write`;
- package GHCR non collegato al repository o non pubblico.
