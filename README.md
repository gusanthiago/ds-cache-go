# Cache Go

This is an application to study and understand Cache strategies in go using an API

Some keys are hashed onto one of N data nodes, so each node stores its keys in a
tiered L1/L2 LRU cache.

### Architecture

The code is split so each layer only talks to the one below it through an
interface, never to a concrete type:

```
HTTP        api/                  bind, validate, map results to status codes
  |
Domain      internal/service/     which node owns a key
  |
Storage     internal/cache/       Store interface
              |- lru/             one LRU tier
              `- tiered/          L1/L2 stack, itself a Store
```

A node is just a named `cache.Store`, so the service holds `[]cache.Store` and
never knows how a node is built. Because `tiered.Cache` implements the same
interface as `lru.Cache`, the layers above cannot tell a stack from a single
tier — adding an L3, or swapping a tier for a Redis-backed store, is a change in
`main.go` wiring only.

#### How the tiers behave

Each node runs a small L1 in front of a full-size L2. The stack is inclusive:

* **Read** — tiers are probed fastest-first. A hit in L2 is promoted into L1, so
  the second read of a key is served from the hot tier.
* **Write** — written through to every tier, which keeps a read straight after a
  write on the fast path.
* **Delete** — removed from every tier, so no stale copy survives in a slower
  layer and gets promoted back up later.

An L1 eviction costs only the extra hop down to L2, not the entry itself.

### How to run

To run application make sure that configurations are what you expected in .env file like sample below

```
CACHE_MANAGER_PORT=":8882"
CACHE_MANAGER_PSEUDO_NODES=3
CACHE_MANAGER_NODES_CAPACITY=1000
CACHE_MANAGER_L1_CAPACITY=10
CACHE_MANAGER_LOG_LEVEL="info"
```

| Variable | Default | Meaning |
| --- | --- | --- |
| `CACHE_MANAGER_PORT` | `:8882` | Listen address; `8882` and `:8882` both work |
| `CACHE_MANAGER_PSEUDO_NODES` | `3` | Number of data nodes in the cluster |
| `CACHE_MANAGER_NODES_CAPACITY` | `1000` | L2 capacity per node, i.e. keys a node can hold |
| `CACHE_MANAGER_L1_CAPACITY` | capacity / 10 | Hot-tier capacity per node |
| `CACHE_MANAGER_LOG_LEVEL` | `info` | `debug`, `info`, `warn` or `error` |

Every value has a default, so a missing `.env` is fine. Invalid configuration is
rejected at startup rather than surfacing later as a misbehaving cache.

Run application

```
go run .
```

Run the tests

```
go test -race ./...
```

### Requests

* Create cache key — `201` when the key is new, `200` when it replaces an existing value

```
curl --location 'http://localhost:8882/api/cache-manager/v1' \
--header 'Content-Type: application/json' \
--data '{
    "key": "test",
    "value": "value-test"
}'
```

```json
{
    "code": 201,
    "status": "Created",
    "data": {
        "key": "test",
        "value": "value-test"
    }
}
```

An invalid body is answered with `400`:

```json
{
    "code": 400,
    "status": "Bad Request",
    "error": "Key: 'CacheEntryRequest.Value' Error:Field validation for 'Value' failed on the 'required' tag"
}
```

* Get cache key

```
curl --location 'http://localhost:8882/api/cache-manager/v1?cacheKey=test'
```

```json
{
    "code": 200,
    "status": "OK",
    "data": {
        "key": "test",
        "value": "value-test"
    }
}
```

A key held by no node is answered with `404`:

```json
{
    "code": 404,
    "status": "Not Found",
    "error": "cache key not found"
}
```

* Delete cache key — `404` when absent

```
curl --location --request DELETE 'http://localhost:8882/api/cache-manager/v1?cacheKey=test'
```

```json
{
    "code": 200,
    "status": "OK",
    "data": {
        "key": "test"
    }
}
```

* Per-node, per-layer statistics — hits, misses, evictions, size and capacity for
  the stack and each tier beneath it

```
curl --location 'http://localhost:8882/api/cache-manager/v1/stats'
```

The sample below comes from one node with `CACHE_MANAGER_L1_CAPACITY=1`, after
writing two keys and then reading the first one twice. The first read missed L1,
hit L2 and was promoted, so the second read was an L1 hit. The two L1 evictions
cost nothing, since L2 still held both keys.

```json
{
    "code": 200,
    "status": "OK",
    "data": [
        {
            "node": "node-1",
            "layers": [
                {
                    "name": "node-1",
                    "hits": 2,
                    "misses": 1,
                    "evictions": 2,
                    "size": 2,
                    "capacity": 8
                },
                {
                    "name": "node-1-l1",
                    "hits": 1,
                    "misses": 2,
                    "evictions": 2,
                    "size": 1,
                    "capacity": 1
                },
                {
                    "name": "node-1-l2",
                    "hits": 1,
                    "misses": 1,
                    "evictions": 0,
                    "size": 2,
                    "capacity": 8
                }
            ]
        }
    ]
}
```

* Health check

```
curl --location 'http://localhost:8882/health'
```

```json
{
    "code": 200,
    "status": "OK",
    "data": {
        "status": "ok"
    }
}
```
