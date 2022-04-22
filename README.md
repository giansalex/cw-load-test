# cw-load-test

`cw-load-test` is a tool for load
testing Cosmwasm networks based on [tm-load-test](https://github.com/informalsystems/tm-load-test).

## Requirements
In order to build and use the tools, you will need:

* Go 1.17+
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
```

**1. Storing wasm code**

This command generates different wasm files from a custom wat (text format) using [wasmtime-go](https://github.com/bytecodealliance/wasmtime-go), 
and sends 6 wasm codes with a timeout height (+5).

```bash
./build/cw-load-test -b 5 -r 6 \
    --wat-path res/code.wat \
    --broadcast-tx-method async \
    --lcd http://127.0.0.1:1317 \
    --endpoints ws://127.0.0.1:26657/websocket \
    --gas 2000000 --gas-prices 0.00ucosm
```

- `-b`: Max block to wait txs complete
- `-r`: Txs in batch transaction

**2. Execute contract**

Send 10 `MsgExecuteContract` txs with timeout height +5. 

```bash
./build/cw-load-test -b 5 -r 10 \
    --contract wasm1hm4y6fzgxgu688jgf7ek66px6xkrtmn3gyk8fax3eawhp68c2d5qphe2pl
    --exec-msg '{"loop": {}}' \
    --broadcast-tx-method async \
    --lcd http://127.0.0.1:1317 \
    --endpoints ws://127.0.0.1:26657/websocket \
    --gas 2000000 --gas-prices 0.00ucosm
```

- `--contract`: Contract to be executed.
- `--exec-msg`: Execute msg to send to contract.
