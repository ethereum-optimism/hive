#!/bin/sh
set -exu

RPC_ADDR=0.0.0.0
RPC_API=eth,rollup,net,web3,debug,miner

L2GETH_GENESIS_PATH=/genesis_goerli.json
L2GETH_GENESIS_HASH=0x1067d2037744f17d34e3ceb88b0d654a3798f5d12b79b348085f13f1ec458636
SYNC_SOURCE=l1
BLOCK_SIGNER_ADDRESS=0x27770a9694e4B4b1E130Ab91Bc327C36855f612E
NODE_TYPE=full

GETH_DATA_DIR=/geth
GETH_CHAINDATA_DIR=$GETH_DATA_DIR/geth/chaindata
GETH_KEYSTORE_DIR=$GETH_DATA_DIR/keystore
BLOCK_SIGNER_PRIVATE_KEY=da5deb73dbc9dea2e3916929daaf079f75232d32a2cf37ff8b1f7140ef3fd9db
BLOCK_SIGNER_PRIVATE_KEY_PASSWORD="pwd"

mkdir $GETH_DATA_DIR

if [ ! -d "$GETH_KEYSTORE_DIR" ]; then
    echo "$GETH_KEYSTORE_DIR missing, running account import"
    echo -n "$BLOCK_SIGNER_PRIVATE_KEY_PASSWORD" > "$GETH_DATA_DIR"/password
    echo -n "$BLOCK_SIGNER_PRIVATE_KEY" > "$GETH_DATA_DIR"/block-signer-key
    geth account import \
        --datadir="$GETH_DATA_DIR" \
        --password="$GETH_DATA_DIR"/password \
        "$GETH_DATA_DIR"/block-signer-key
    echo "get account import complete"
fi

if [ ! -d "$GETH_CHAINDATA_DIR" ]; then
    echo "$GETH_CHAINDATA_DIR missing, running init"
    geth init --datadir="$GETH_DATA_DIR" "$L2GETH_GENESIS_PATH" "$L2GETH_GENESIS_HASH"
    echo "geth init complete"
else
    echo "$GETH_CHAINDATA_DIR exists, checking for hardfork."
    echo "Chain config:"
    geth dump-chain-cfg --datadir="$GETH_DATA_DIR"
fi

# Set rollup backend to match sync source
export ROLLUP_BACKEND=$SYNC_SOURCE

# Run geth
exec geth \
  --vmodule=eth/*=5,miner=4,rpc=5,rollup=4,consensus/clique=1 \
  --datadir=$GETH_DATA_DIR \
  --password=$GETH_DATA_DIR/password \
  --allow-insecure-unlock \
  --unlock=$BLOCK_SIGNER_ADDRESS \
  --mine \
  --miner.etherbase=$BLOCK_SIGNER_ADDRESS \
  --gcmode=$NODE_TYPE \
  --rpc \
	--rpcaddr $RPC_ADDR \
	--mine \
	--miner.etherbase $BLOCK_SIGNER_ADDRESS \
	--rpcapi $RPC_API \
  $@
