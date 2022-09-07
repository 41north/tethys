module "ethereum_nats" {
  source = "../../../modules/ethereum-nats"

  network_id = var.network_id
  chain_id = var.chain_id
}