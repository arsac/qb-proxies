# qb-proxies

A lightweight RSS feed proxy that transforms non-standard torrent RSS feeds into a format compatible with qBittorrent and other torrent clients.

## Features

- Proxy and transform RSS feeds on-the-fly
- CEL (Common Expression Language) for flexible field transformations
- Fix missing or malformed enclosure tags
- Convert infohash to magnet links
- Docker support with GitHub Actions CI/CD

## Installation

### Docker

```bash
docker run -v /path/to/config.yaml:/config/config.yaml -p 8080:8080 ghcr.io/arsac/qb-proxies
```

### From source

```bash
go build -o qb-proxies .
./qb-proxies -config config.yaml -addr :8080
```

## Configuration

Create a `config.yaml` file (see `config.example.yaml` for reference):

```yaml
feeds:
  - name: academic-torrents
    path: /academic
    upstream: https://academictorrents.com/rss.xml
    transformations:
      - field: enclosure.url
        expression: '"magnet:?xt=urn:btih:" + infohash'
      - field: enclosure.length
        expression: 'size'
      - field: enclosure.type
        expression: '"application/x-bittorrent"'
```

### Feed options

| Field | Description |
|-------|-------------|
| `name` | Unique identifier for the feed |
| `path` | URL path to expose (e.g., `/academic`) |
| `upstream` | Source RSS feed URL |
| `transformations` | List of CEL transformations to apply |

### Transformation fields

| Field | Description |
|-------|-------------|
| `title` | Item title |
| `link` | Item link |
| `description` | Item description |
| `guid` | Item GUID |
| `pubDate` | Publication date |
| `enclosure.url` | Enclosure URL (torrent/magnet link) |
| `enclosure.length` | Enclosure size in bytes |
| `enclosure.type` | Enclosure MIME type |

### CEL variables

Standard RSS fields: `title`, `link`, `description`, `guid`, `pubDate`, `enclosureUrl`, `enclosureLength`, `enclosureType`

Non-standard fields (for feeds like Academic Torrents): `infohash`, `size`, `category`

### CEL examples

```yaml
# Build magnet link from infohash
- field: enclosure.url
  expression: '"magnet:?xt=urn:btih:" + infohash'

# Use link as fallback for missing enclosure
- field: enclosure.url
  expression: 'enclosureUrl != "" ? enclosureUrl : link'

# Prefix titles
- field: title
  expression: '"[Proxy] " + title'

# Strip query parameters from URLs
- field: link
  expression: 'link.split("?")[0]'
```

## Usage with qBittorrent

1. Start the proxy with your config
2. In qBittorrent, go to RSS > New subscription
3. Add the proxy URL: `http://localhost:8080/academic`

## Endpoints

- `GET /{path}` - Proxied and transformed RSS feed
- `GET /healthz` - Kubernetes health check
- `GET /readyz` - Kubernetes readiness probe
- `GET /livez` - Kubernetes liveness probe

## License

MIT
