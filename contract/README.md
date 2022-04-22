# Cw load

Contract based on [cosmwasm example](https://github.com/CosmWasm/cosmwasm/blob/main/contracts/hackatom/).

## Messages

Allocate large amounts of memory:

```json
{
  "allocate_memory": {
    "pages": 200
  }
}
```

Make storage calls
```json
{
  "storage_loop": {
    "prefix": "test",
    "data": "xxxxxxxxx_base64_xxxxxxxxxxxx==",
    "limit": 100
  }
}
```

Cpu loops
```json
{
  "cpu_loop": {
    "limit": 200000000
  }
}
```
