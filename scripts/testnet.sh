#!/bin/bash

# Single-node local testnet for webstackd.
# Usage: ./scripts/testnet.sh [-y]
#   -y   Overwrite existing chain data without prompt

set -e

BINARY="webstackd"
CHAINID="webstack"
EVM_CHAINID=262144
MONIKER="webstack-testnode"
KEYRING="test"
KEYALGO="eth_secp256k1"
LOGLEVEL="info"
CHAINDIR="$HOME/.webstackd"
DENOM="aatom"
BASEFEE=10000000

CONFIG_TOML="$CHAINDIR/config/config.toml"
APP_TOML="$CHAINDIR/config/app.toml"
GENESIS="$CHAINDIR/config/genesis.json"
TMP_GENESIS="$CHAINDIR/config/tmp_genesis.json"

# ---------- Dependency checks ----------
command -v jq >/dev/null 2>&1 || { echo >&2 "jq is required but not installed. Install it: brew install jq"; exit 1; }
command -v "$BINARY" >/dev/null 2>&1 || { echo >&2 "$BINARY not found in PATH. Run 'make install' first or let 'make sh-testnet' handle it."; exit 1; }

# ---------- Flags ----------
overwrite=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    -y) overwrite="y"; shift ;;
    -n) overwrite="n"; shift ;;
    *) echo "Unknown flag: $1"; exit 1 ;;
  esac
done

# Prompt if data dir already exists and no flag was given
if [[ -z "$overwrite" ]]; then
  if [ -d "$CHAINDIR" ]; then
    printf "\nExisting chain data found at '%s'.\nOverwrite and start fresh? [y/n] " "$CHAINDIR"
    read -r overwrite
  else
    overwrite="y"
  fi
fi

# ---------- Setup ----------
if [[ "$overwrite" == "y" || "$overwrite" == "Y" ]]; then
  echo "==> Removing old chain data..."
  rm -rf "$CHAINDIR"

  echo "==> Configuring client..."
  $BINARY config set client chain-id "$CHAINID" --home "$CHAINDIR"
  $BINARY config set client keyring-backend "$KEYRING" --home "$CHAINDIR"

  # ---------- Validator key ----------
  VAL_KEY="validator"
  VAL_MNEMONIC="gesture inject test cycle original hollow east ridge hen combine junk child bacon zero hope comfort vacuum milk pitch cage oppose unhappy lunar seat"

  echo "==> Adding validator key..."
  echo "$VAL_MNEMONIC" | $BINARY keys add "$VAL_KEY" --recover --keyring-backend "$KEYRING" --algo "$KEYALGO" --home "$CHAINDIR"

  # ---------- Dev accounts ----------
  DEV0_MNEMONIC="copper push brief egg scan entry inform record adjust fossil boss egg comic alien upon aspect dry avoid interest fury window hint race symptom"
  DEV1_MNEMONIC="maximum display century economy unlock van census kite error heart snow filter midnight usage egg venture cash kick motor survey drastic edge muffin visual"

  echo "==> Adding dev accounts..."
  echo "$DEV0_MNEMONIC" | $BINARY keys add dev0 --recover --keyring-backend "$KEYRING" --algo "$KEYALGO" --home "$CHAINDIR"
  echo "$DEV1_MNEMONIC" | $BINARY keys add dev1 --recover --keyring-backend "$KEYRING" --algo "$KEYALGO" --home "$CHAINDIR"

  # ---------- Init chain ----------
  echo "==> Initializing chain..."
  echo "$VAL_MNEMONIC" | $BINARY init "$MONIKER" -o --chain-id "$CHAINID" --home "$CHAINDIR" --recover

  # ---------- Genesis customizations ----------
  echo "==> Customizing genesis..."

  # Set bond denom
  jq '.app_state["staking"]["params"]["bond_denom"]="'"$DENOM"'"' "$GENESIS" > "$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

  # Set gov deposit denom
  jq '.app_state["gov"]["params"]["min_deposit"][0]["denom"]="'"$DENOM"'"' "$GENESIS" > "$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
  jq '.app_state["gov"]["params"]["expedited_min_deposit"][0]["denom"]="'"$DENOM"'"' "$GENESIS" > "$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

  # Set EVM denom
  jq '.app_state["evm"]["params"]["evm_denom"]="'"$DENOM"'"' "$GENESIS" > "$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

  # Set mint denom
  jq '.app_state["mint"]["params"]["mint_denom"]="'"$DENOM"'"' "$GENESIS" > "$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

  # Bank denom metadata
  jq '.app_state["bank"]["denom_metadata"]=[{
    "description": "The native token of the Webstack chain.",
    "denom_units": [
      {"denom": "'"$DENOM"'", "exponent": 0, "aliases": ["attoatom"]},
      {"denom": "atom", "exponent": 18, "aliases": []}
    ],
    "base": "'"$DENOM"'",
    "display": "atom",
    "name": "Atom",
    "symbol": "ATOM",
    "uri": "",
    "uri_hash": ""
  }]' "$GENESIS" > "$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

  # Enable all static precompiles
  jq '.app_state["evm"]["params"]["active_static_precompiles"]=[
    "0x0000000000000000000000000000000000000100",
    "0x0000000000000000000000000000000000000400",
    "0x0000000000000000000000000000000000000800",
    "0x0000000000000000000000000000000000000801",
    "0x0000000000000000000000000000000000000802",
    "0x0000000000000000000000000000000000000803",
    "0x0000000000000000000000000000000000000804",
    "0x0000000000000000000000000000000000000805",
    "0x0000000000000000000000000000000000000806",
    "0x0000000000000000000000000000000000000807"
  ]' "$GENESIS" > "$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

  # ERC20 native precompile & token pair
  jq '.app_state.erc20.native_precompiles=["0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE"]' "$GENESIS" > "$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
  jq '.app_state.erc20.token_pairs=[{
    "contract_owner": 1,
    "erc20_address": "0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE",
    "denom": "'"$DENOM"'",
    "enabled": true
  }]' "$GENESIS" > "$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

  # Block gas limit
  jq '.consensus.params.block.max_gas="10000000"' "$GENESIS" > "$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

  # Shorten gov voting periods for testing
  sed -i.bak 's/"max_deposit_period": "172800s"/"max_deposit_period": "30s"/g' "$GENESIS"
  sed -i.bak 's/"voting_period": "172800s"/"voting_period": "30s"/g' "$GENESIS"
  sed -i.bak 's/"expedited_voting_period": "86400s"/"expedited_voting_period": "15s"/g' "$GENESIS"

  # ---------- Fund accounts ----------
  echo "==> Funding genesis accounts..."
  FUND_AMOUNT="100000000000000000000000000${DENOM}"
  $BINARY genesis add-genesis-account "$VAL_KEY" "$FUND_AMOUNT" --keyring-backend "$KEYRING" --home "$CHAINDIR"
  $BINARY genesis add-genesis-account dev0 "1000000000000000000000${DENOM}" --keyring-backend "$KEYRING" --home "$CHAINDIR"
  $BINARY genesis add-genesis-account dev1 "1000000000000000000000${DENOM}" --keyring-backend "$KEYRING" --home "$CHAINDIR"

  # ---------- CometBFT config tweaks ----------
  echo "==> Tuning CometBFT config for fast blocks..."
  sed -i.bak 's/timeout_propose = "3s"/timeout_propose = "2s"/g' "$CONFIG_TOML"
  sed -i.bak 's/timeout_propose_delta = "500ms"/timeout_propose_delta = "200ms"/g' "$CONFIG_TOML"
  sed -i.bak 's/timeout_prevote = "1s"/timeout_prevote = "500ms"/g' "$CONFIG_TOML"
  sed -i.bak 's/timeout_prevote_delta = "500ms"/timeout_prevote_delta = "200ms"/g' "$CONFIG_TOML"
  sed -i.bak 's/timeout_precommit = "1s"/timeout_precommit = "500ms"/g' "$CONFIG_TOML"
  sed -i.bak 's/timeout_precommit_delta = "500ms"/timeout_precommit_delta = "200ms"/g' "$CONFIG_TOML"
  sed -i.bak 's/timeout_commit = "5s"/timeout_commit = "1s"/g' "$CONFIG_TOML"
  sed -i.bak 's/timeout_broadcast_tx_commit = "10s"/timeout_broadcast_tx_commit = "5s"/g' "$CONFIG_TOML"

  # Enable prometheus
  sed -i.bak 's/prometheus = false/prometheus = true/' "$CONFIG_TOML"

  # Set EVM chain ID in app.toml
  sed -i.bak "s/evm-chain-id = .*/evm-chain-id = ${EVM_CHAINID}/" "$APP_TOML"

  # Enable APIs and Swagger
  sed -i.bak 's/prometheus-retention-time  = "0"/prometheus-retention-time  = "1000000000000"/g' "$APP_TOML"
  sed -i.bak 's/enabled = false/enabled = true/g' "$APP_TOML"
  sed -i.bak 's/enable = false/enable = true/g' "$APP_TOML"
  sed -i.bak 's/swagger = false/swagger = true/g' "$APP_TOML"

  # Clean up .bak files from sed
  find "$CHAINDIR" -name "*.bak" -delete

  # ---------- Create gentx & finalize ----------
  echo "==> Creating gentx..."
  $BINARY genesis gentx "$VAL_KEY" "1000000000000000000000${DENOM}" \
    --gas-prices "${BASEFEE}${DENOM}" \
    --keyring-backend "$KEYRING" \
    --chain-id "$CHAINID" \
    --home "$CHAINDIR"

  echo "==> Collecting gentxs..."
  $BINARY genesis collect-gentxs --home "$CHAINDIR"

  echo "==> Validating genesis..."
  $BINARY genesis validate-genesis --home "$CHAINDIR"
fi

# ---------- Start ----------
echo ""
echo "============================================"
echo "  Starting webstackd testnet"
echo "  Cosmos Chain ID:  $CHAINID"
echo "  EVM Chain ID:     $EVM_CHAINID"
echo "  RPC:        http://localhost:26657"
echo "  REST API:   http://localhost:1317"
echo "  gRPC:       localhost:9090"
echo "  EVM RPC:    http://localhost:8545"
echo "============================================"
echo ""

$BINARY start \
  --pruning nothing \
  --log_level "$LOGLEVEL" \
  --minimum-gas-prices "0${DENOM}" \
  --evm.min-tip=0 \
  --home "$CHAINDIR" \
  --json-rpc.api eth,txpool,personal,net,debug,web3 \
  --chain-id "$CHAINID"
