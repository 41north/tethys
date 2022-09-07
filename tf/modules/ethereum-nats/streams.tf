resource "jetstream_stream" "eth_stream_newHeads" {
  name     = "eth_${var.network_id}_${var.chain_id}_newHeads"
  subjects = ["eth.${var.network_id}.${var.chain_id}.newHeads.>"]
  storage  = "file"
  max_age  = 60 * 60 * 24 * 14
}