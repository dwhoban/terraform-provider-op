ephemeral "op_secret_ref" "db_password" {
  reference = "op://Infrastructure/Production database/password"
}

# The resolved value never persists to state or plan; use it in write-only
# attributes, provider config, or provisioners.
