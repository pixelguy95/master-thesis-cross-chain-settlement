#!/bin/bash

# Remote bitcoind RPC client through SSH tunnel
#
# found this great resource https://gist.github.com/EnigmaCurry/bdd9fd28d7a73fe52eb4/revisions
#
# If you have a remote bitcoind you'd like to query it's RPC interface
# from, this script will help you maintain the SSH tunnel to do so.
#
# Set REMOTE_HOST to the user@server your remote bitcoind is running on
# Set LOCAL_FORWARD_PORT to the port you want to run the tunnel on
# Set the RPC_PASSWORD to the rpc password bitcoind is set to use.
# Install bitcoind locally, so that you can use it's query interface.
#
# The first run will setup the SSH tunnel, and leave it running. 
# Subsequent runs will be fast.

# Set port for mainnet

REMOTE_HOST=pi@bitcoin.ndersson.io
LOCAL_FORWARD_PORT=8332

# Check if the tunnel is already open:
port_open=$(netstat -lnt | grep 127.0.0.1:$LOCAL_FORWARD_PORT | wc -l)
if [ $port_open -lt 1 ] 
then
  echo "Creating port forward..."
  ssh -N $REMOTE_HOST -L $LOCAL_FORWARD_PORT:localhost:8332 &
  sleep 5

  # Check if the tunnel was created successfully:
  port_open=$(netstat -lnt | grep 127.0.0.1:$LOCAL_FORWARD_PORT | wc -l)
  if [ $port_open -lt 1 ] 
  then
    echo "Could not create port forward"
    exit 1
  fi
fi

# Set port for testnet

LOCAL_FORWARD_PORT=18332
# Check if the tunnel is already open:
port_open=$(netstat -lnt | grep 127.0.0.1:$LOCAL_FORWARD_PORT | wc -l)
if [ $port_open -lt 1 ]
then
  echo "Creating port forward..."
  ssh -N $REMOTE_HOST -L $LOCAL_FORWARD_PORT:localhost:18332 &
  sleep 5

  # Check if the tunnel was created successfully:
  port_open=$(netstat -lnt | grep 127.0.0.1:$LOCAL_FORWARD_PORT | wc -l)
  if [ $port_open -lt 1 ]
  then
    echo "Could not create port forward"
    exit 1
  fi
fi

#bitcoind -rpcport=$LOCAL_FORWARD_PORT -rpcpassword=$RPC_PASSWORD $*