# Distributed Cache Go

### How to run

To run application make sure that configurations are what you expected in .env file like sample below

```
CACHE_MANAGER_PORT=":8882"
CACHE_MANAGER_NODES_NUMBER=3
CACHE_MANAGER_NODES_CAPACITY=1000
```

Run application

```
go run .
```

### Requests 

* Create cache key

```
curl --location 'http://localhost:8882/api/cache-manager/v1' \
--header 'Content-Type: application/json' \
--data '{
    "key": "test",
    "value": "value-test"
}'
```

* Get cache key

```
curl --location 'http://localhost:8882/api/cache-manager/v1?cacheKey=test'
```




### TODO

Some todo points that couldn't be done in time

* Implement pub/sub mecanism to connect datanode with API
* Improve concurrency of cache system
* Expand test coverage
