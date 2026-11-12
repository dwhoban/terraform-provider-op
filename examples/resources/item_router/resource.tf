resource "op_item_router" "router" {
  vault_id              = op_vault.example.id
  title                 = "Router example"
  base_station_name     = "Living Room"
  base_station_password = "admin"
  network_name          = "Appleseed 5G"
}
