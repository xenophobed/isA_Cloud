# Consul Production Configuration
# Replace consul agent -dev with this configuration

datacenter = "isa-cloud-dc1"
data_dir = "/tmp/consul/data"
log_level = "INFO"
server = true
bootstrap_expect = 1

# Network Configuration
bind_addr = "127.0.0.1"
client_addr = "127.0.0.1"

# UI Configuration
ui_config {
  enabled = true
}

# Performance
performance {
  raft_multiplier = 1
}

# Connect (Service Mesh)
connect {
  enabled = true
}

# Security Configuration
acl = {
  enabled = true
  default_policy = "allow"  # Change to "deny" in production
  enable_token_persistence = true
}

# Ports Configuration
ports {
  grpc = 8502
  grpc_tls = 8503
}

# Telemetry
telemetry {
  prometheus_retention_time = "24h"
  disable_hostname = true
}