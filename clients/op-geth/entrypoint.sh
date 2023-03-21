#!/bin/sh
set -exu

# note: geth wants an integer log level (see L1 hive definition)
VERBOSITY=${HIVE_LOGLEVEL:-3}

GETH_DATA_DIR=/db
GETH_CHAINDATA_DIR="$GETH_DATA_DIR/geth/chaindata"

CHAIN_ID=$(cat /genesis.json | jq -r .config.chainId)

if [ ! -d "$GETH_CHAINDATA_DIR" ]; then
	echo "$GETH_CHAINDATA_DIR missing, running init"
	echo "Initializing genesis."
	geth --verbosity="$VERBOSITY" init \
		--datadir="$GETH_DATA_DIR" \
		"/genesis.json"
else
	echo "$GETH_CHAINDATA_DIR exists."
fi

# We must set miner.gaslimit to the gas limit in genesis
# in the command below!
GAS_LIMIT_HEX=$(jq -r .gasLimit < /genesis.json | sed s/0x//i | tr '[:lower:]' '[:upper:]')
GAS_LIMIT=$(echo "obase=10; ibase=16; $GAS_LIMIT_HEX" | bc)


EXTRA_FLAGS="--rollup.disabletxpoolgossip=true"

# We check for env variables that may not be bound so we need to disable `set -u` for this section.
set +u
if [ "$HIVE_OP_GETH_SEQUENCER_HTTP" != "" ]; then
  EXTRA_FLAGS="$EXTRA_FLAGS --rollup.sequencerhttp=$HIVE_OP_GETH_SEQUENCER_HTTP"
fi
if [ "$HIVE_OP_GETH_HISTORICAL_RPC" != "" ]; then
	EXTRA_FLAGS="$EXTRA_FLAGS --rollup.historicalrpc=$HIVE_OP_GETH_HISTORICAL_RPC"
fi
if [ "$HIVE_OP_GETH_USE_GOERLI_DATADIR" != "" ]; then
	curl -L -o /goerli-bedrock.tar https://storage.googleapis.com/oplabs-goerli-data/goerli-bedrock.tar
	mkdir /goerli-bedrock
	tar -xvf /goerli-bedrock.tar -C /goerli-bedrock
	GETH_DATA_DIR="/goerli-bedrock"
fi
set -u

# Warning: Archive mode is required, otherwise old trie nodes will be
# pruned within minutes of starting the devnet.

geth \
	--datadir="$GETH_DATA_DIR" \
	--verbosity="$VERBOSITY" \
	--http \
	--http.corsdomain="*" \
	--http.vhosts="*" \
	--http.addr=0.0.0.0 \
	--http.port=8545 \
	--http.api=web3,debug,eth,txpool,net,engine \
	--ws \
	--ws.addr=0.0.0.0 \
	--ws.port=8546 \
	--ws.origins="*" \
	--ws.api=debug,eth,txpool,net,engine \
	--authrpc.jwtsecret="/hive/input/jwt-secret.txt" \
	--authrpc.port=8551 \
	--authrpc.addr=0.0.0.0 \
	--syncmode=full \
	--nodiscover \
	--maxpeers=0 \
	--miner.gaslimit=$GAS_LIMIT \
	--networkid="$CHAIN_ID" \
	--password="$GETH_DATA_DIR"/password \
	--allow-insecure-unlock \
	--gcmode=archive \
	$EXTRA_FLAGS \
	"$@"
