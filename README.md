# issuer

Terminal-UI für Issues aus verschiedenen Systemen. Links stehen alle Projekte, rechts die Issues des gewählten Projekts. Issues lassen sich öffnen, kommentieren, mit Labels versehen, schließen und neu anlegen.

Unterstützt wird bisher **GitHub**. Weitere Systeme werden über das Interface in `internal/provider/provider.go` angebunden.

## Installation

```bash
go install github.com/ljunker/issuer/cmd/issuer@latest
```

Oder lokal bauen:

```bash
go build -o issuer ./cmd/issuer
```

## Anmeldung

Der GitHub-Token wird in dieser Reihenfolge gesucht:

1. `GITHUB_TOKEN`
2. `GH_TOKEN`
3. `gh auth token` (GitHub CLI)

Für GitHub Enterprise kann die API-URL über `ISSUER_GITHUB_API_URL` gesetzt werden.

## Konfiguration (optional)

`~/.config/issuer/config.toml`:

```toml
# Diese Projekte stehen oben in der Liste
pinned = ["owner/repo", "org/anderes-repo"]

# Status-Filter beim Start: open, closed oder all
default_state = "open"
```

## Tasten

| Taste | Aktion |
| --- | --- |
| `↑`/`↓`, `j`/`k` | navigieren |
| `tab` | Fokus zwischen Projekten und Issues wechseln |
| `enter` | Projekt bzw. Issue öffnen |
| `esc` | zurück / Filter entfernen |
| `/` | Liste filtern |
| `n` | neues Issue im aktuellen Projekt |
| `c` | Issue kommentieren |
| `l` | Labels verwalten (`space` umschalten, `/` suchen, `enter` übernehmen) |
| `x` | Issue schließen bzw. wieder öffnen (mit Rückfrage) |
| `s` | Status-Filter: offen → geschlossen → alle |
| `r` | Issues (und Kommentare) neu laden |
| `R` | Projektliste neu laden |
| `o` | Projekt bzw. Issue im Browser öffnen |
| `ctrl+s` | Formular absenden |
| `?` | alle Tasten anzeigen |
| `q` | beenden |

## Aufbau

```
cmd/issuer/              Einstieg
internal/provider/       systemneutrales Interface und Modelle
internal/provider/github GitHub-Anbindung (go-github)
internal/auth            Token-Suche
internal/config          Konfigurationsdatei
internal/ui              Bubble-Tea-Oberfläche
```
