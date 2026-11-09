data "op_vaults" "all" {}

# output "titles" { value = [for v in data.op_vaults.all.vaults : v.title] }
