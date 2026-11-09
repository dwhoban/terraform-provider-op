data "onepassword_vaults" "all" {}

# output "titles" { value = [for v in data.onepassword_vaults.all.vaults : v.title] }
