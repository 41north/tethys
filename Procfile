# Procfile - https://devcenter.heroku.com/articles/procfile

# Prysm / Sepolia / https://docs.prylabs.network/docs/install/install-with-script
prysm: beacon-chain --accept-terms-of-use --sepolia --datadir="${PRYSM_DATA}" --rpc-host="0.0.0.0" --execution-endpoint="${GETH_IPC}" --monitoring-host="0.0.0.0" --grpc-gateway-host="0.0.0.0" --grpc-gateway-corsdomain="*" --checkpoint-sync-url="${PRYSM_CHECKPOINT_URL}" --genesis-beacon-api-url="${PRYSM_CHECKPOINT_URL}" --genesis-state="${PRYSM_CHECKPOINT}"

# Geth / Sepolia
geth: geth --sepolia --datadir="${GETH_DATA}" --http --http.api eth,net,engine,admin --ws --override.terminaltotaldifficulty 17000000000000000

# NATS / Server / https://docs.nats.io/running-a-nats-service/introduction/running
nats: nats-server --addr="0.0.0.0" --port=4222 --http_port=8222 --jetstream --store_dir="${NATS_DATA}"