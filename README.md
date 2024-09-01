# go lgtm sample

query example
```
curl -X"POST" -H"content-type: application/json" localhost:8000/calc -d '{"op": "/", "a": 30, "b": {"op": "-", "a": 10, "b": 10}}' -i
```

start k6
```
docker compose run --rm k6 run /scripts/loadtest.js
```
