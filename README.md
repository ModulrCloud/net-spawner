# NetSpawner

**NetSpawner** is a small Go utility for launching and managing a local multi-node blockchain test network.  
It prepares per-node directories, copies configs/genesis files, optionally resets on-disk state, and starts the node binary for each validator. All node logs are streamed to your console with a per-node prefix.

Use cases:
- Local development and integration testing
- Spinning up networks with a chosen validator count
- Fast environment resets after core changes

---

## Build

Requirements: Go 1.21+

```bash
git clone https://github.com/modulrcloud/net-spawner.git

cd net-spawner

go build -o netspawner main.go
```

## Usage

NetSpawner reads settings from `configs.json` in the same directory as the binary

Example config:

```json
{
  "corePath": "./bin/core-node",         // path to your Go node binary
  "netMode": "TESTNET_2V"                // possible variants are TESTNET_1V | TESTNET_2V | TESTNET_5V
}
```

Start commands:

```bash
# Resume network using existing on-disk data
./netspawner resume

# Full reset (recreate dirs/files, update genesis timestamp, wipe CHAINDATA), then start
./netspawner reset

# Generate a new Ed25519 key pair as JSON (optionally pass an existing mnemonic/password)
./netspawner keygen -mnemonic "word1 ... word24" -passphrase "secret" -path 44/7337/0/0
```

Log output: each node runs as a separate process. stdout/stderr are streamed and prefixed, e.g.

```
[./XTESTNET_V5/V1]: Starting node...
[./XTESTNET_V5/V2]: Connected to peers
```



## Commands: resume vs reset

##### resume
Launches all nodes with the existing data folders (CHAINDATA, configs, genesis).
Best when you want to continue from the current state.

##### reset
Rebuilds the per-node directory structure, copies fresh configs, and sets FIRST_EPOCH_START_TIMESTAMP in each genesis.json to the current time.
Wipes CHAINDATA and then automatically runs resume.
Best when you need a clean start from genesis.


## Network size options

##### Single validator (TESTNET_1V)

For quick tests of epoch rotation, block approvals, and overall behavior.
TL;DR — when you need fast checks for database interactions, endpoint routes, networking, and general logic.

##### Two validators (TESTNET_2V)
For testing leader rotation, required proofs behavior, and ensuring serialization correctness across multiple validators.

##### Five validators (TESTNET_5V)
Same goals as the two-validator setup, but with a larger validator set to exercise more complex conditions and higher message/approval volume.


# Usage with PM2 (process manager)

![alt text](files/images/pm2.png)