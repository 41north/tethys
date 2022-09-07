resource "jetstream_kv_bucket" "eth_kv_client_profile" {
  name = "eth_${var.network_id}_${var.chain_id}_client_profile"
}

resource "jetstream_kv_bucket" "eth_kv_client_status" {
  name = "eth_${var.network_id}_${var.chain_id}_client_status"
  history = 24
}

resource "jetstream_kv_bucket" "eth_kv_proxy_responses" {
  name = "eth_${var.network_id}_${var.chain_id}_proxy_responses"
  ttl = 60 * 60 // 1 hour
}