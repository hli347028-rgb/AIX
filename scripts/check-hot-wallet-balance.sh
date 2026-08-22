#!/bin/bash
set -e
RPC=https://rpc1.eoeo.info
WIN=0x193013574dacbd38bf26ecb654b3fd787b94d216
TX=0x5794a0394cf63cfc86b4f46fe6473fe2a1f6051bfd8d2775f0c6fa46304b469f
ADDR=$(curl -sS -X POST "$RPC" -H 'Content-Type: application/json' \
  -d "{\"jsonrpc\":\"2.0\",\"method\":\"eth_getTransactionByHash\",\"params\":[\"$TX\"],\"id\":1}" \
  | python3 -c "import sys,json; t=json.load(sys.stdin).get('result'); print((t or {}).get('from',''))")
echo "hot_wallet: $ADDR"
if [ -z "$ADDR" ]; then exit 0; fi
GAS=$(curl -sS -X POST "$RPC" -H 'Content-Type: application/json' \
  -d "{\"jsonrpc\":\"2.0\",\"method\":\"eth_getBalance\",\"params\":[\"$ADDR\",\"latest\"],\"id\":1}" \
  | python3 -c "import sys,json; print(int(json.load(sys.stdin)['result'],16)/1e18)")
DATA="0x70a08231$(printf '%064s' ${ADDR#0x} | tr ' ' 0)"
WINBAL=$(curl -sS -X POST "$RPC" -H 'Content-Type: application/json' \
  -d "{\"jsonrpc\":\"2.0\",\"method\":\"eth_call\",\"params\":[{\"to\":\"$WIN\",\"data\":\"$DATA\"},\"latest\"],\"id\":1}" \
  | python3 -c "import sys,json; r=json.load(sys.stdin).get('result','0x0'); print(int(r,16)/1e18)")
echo "native_gas_balance: $GAS"
echo "win_token_balance: $WINBAL"
