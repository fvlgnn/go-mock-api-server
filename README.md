# go-mock-api-server

Mock API Server Go. Genera API fittizie configurabili tramite file JSON per test e sviluppo.


## Descrizione

Server di simulazione API personalizzabile. Genera risposte API personalizzabili e con dati fittizi, configurabili tramite file JSON, ideali per lo sviluppo e il testing di applicazioni. Scritto in Go.

Nella configurazione degli endpoint è possibile impostare il path, il motodo HTTP e la risposta .


## Configurazione

Aggiungere nella cartella `config` i file JSON con le configurazioni dei singoli endpoint.

Esempio:

```json
{
  "request": {
    "method": "GET",
    "path": "/v1/user"
  },
  "response": {
    "body": [
      { "id": 1, "name": "Tom" }
    ]
  }
}
```

In questo caso:

- Il **Metodo HTTP** accettato è `GET`
- L'**Endpoint**, il _path_ dove puntare la richiesta è `/v1/user`
- La **Risposta** sarà `[{"id": 1,"name": "Tom"}]`

Altri esempi di file JSON già configurati sono presenti all'interno della cartella `config`.

| File                  | Path                  | Method    |
|:----------------------|:----------------------|:----------|
| `get_users.json`      | `/v1/users`           | `GET`     |
| `get_user.json`       | `/v1/user/4`          | `GET`     |
| `create_user.json`    | `/v1/user/create`     | `POST`    |
| `delete_user.json`    | `/v1/user/delete/3`   | `DELETE`  |


### ATTENZIONE

**NON utilizzare file che contengono lo stesso endpoint `path` per non creare un _panic_ dell'applicazione Go.**


## Esecuzione

### Locale

```sh
go run main.go
```

### Container

#### Build

```sh
docker build -t go-mock-api-server .
```

#### Run

```sh
docker run -it --rm --name go-mock-api-server -p 8080:8080 go-mock-api-server
# oppure
docker run -it -d --name go-mock-api-server -p 8080:8080 go-mock-api-server
```

- `--rm` cancella il container ed eventuali elementi alla chiusura
- `-p PORTA_HOST:PORTA_APP` (porta host è la porta locale della tua macchina o del server da dove intendi esporre l'applicazione)


### Personalizzazioni

È possibile modificare il nome della cartella e la porta di ascolto del server API dal file sorgente `main.go` dove sono dichiarate le _variabili globali_ del progetto. 

```go
// NOTE Variabili globali
var serverPort = "8080" // porta in ascolto sul server
var configDir = "config" // nome della cartella contenete i file json con le configurazioni dei singoli endpoint
```

Nel caso di un _esecuzione_ tramite _docker strategy_ dovranno essere modificati anche i parametri del comando di `docker run` per l'argomento che imposta il nome della cartella contenente i file di configurazione JSON `CONFIG_DIR` e la porta si ascolto dell'applicazione:

- `docker build --build-arg CONFIG_DIR=config_dir_personalizzata ...` 
- `docker run ... -p PORTA_HOST:PORTA_APP ...`


## TODO

- [x] con docker strategy, aggiungi variabile d'ambiente su dockerfile per impostare il nome della cartella contenente i file JSON di configurazione
- [ ] con docker strategy aggiungi opzione volume sulla cartella contenente i file JSON
- [ ] aggiungi gestione dell'errore nel caso ci siano due endpoint con path identici, possibilmente nella fase di init per evitare il panic.
- [ ] aggiungi su `main.go` una condizione per il quale se impostata la variabile `CONFIG_DIR` modifica la variabile globale `var configDir = "config"` (esempio `var configDir = os.Getenv("CONFIG_DIR")`) in modo da non creare problemi se l'app viene usata sotto container o come standalone

