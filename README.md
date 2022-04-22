# cw-load-test

`cw-load-test` is a tool for load
testing Cosmwasm networks based on [tm-load-test](https://github.com/informalsystems/tm-load-test).

## Requirements
In order to build and use the tools, you will need:

* Go 1.15+
* `make`

## Building
To build the `cw-load-test` binary in the `build` directory:

```bash
make build
```

## Usage

Require `WALLET` and `CHAINID` environment variables.

```bash
# wallet paraphrase
export WALLET="dss hgg ssa yyrre ere ere erre ..."
export CHAINID=torri-1
cw-load-test -b 5 -r 1 \
    --wat-path hack.wat \
    --broadcast-tx-method async \
    --lcd http://135.181.153.131:1317 \
    --endpoints ws://141.95.111.105:26657/websocket \
    --gas 1500000 --gas-prices 0.00utorii
```

- `-b`: Max block to wait txs complete
- `-r`: Txs in batch transaction

Example `-b 8 -r 1000`:

To see a description of what all of the parameters mean, simply run:

```bash
cw-load-test --help
```

## Development
To run the linter and the tests:

```bash
make lint
make test
```

