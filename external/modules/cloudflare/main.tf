data "cloudflare_zones" "zone" {
  name = "meyeringh.org"
}

resource "random_id" "tunnel_secret" {
  byte_length = 32
}

resource "cloudflare_zero_trust_tunnel_cloudflared" "homelab" {
  account_id    = var.cloudflare_account_id
  name          = "homelab"
  tunnel_secret = random_id.tunnel_secret.b64_std
}

resource "cloudflare_dns_record" "tunnel" {
  zone_id = data.cloudflare_zones.zone.result[0].id
  type    = "CNAME"
  name    = "homelab-tunnel"
  content = "${cloudflare_zero_trust_tunnel_cloudflared.homelab.id}.cfargotunnel.com"
  proxied = false
  ttl     = 1 # Auto
}

resource "kubernetes_secret" "cloudflared_credentials" {
  metadata {
    name      = "cloudflared-credentials"
    namespace = "cloudflared"

    annotations = {
      "app.kubernetes.io/managed-by" = "Terraform"
    }
  }

  data = {
    "credentials.json" = jsonencode({
      AccountTag   = var.cloudflare_account_id
      TunnelName   = cloudflare_zero_trust_tunnel_cloudflared.homelab.name
      TunnelID     = cloudflare_zero_trust_tunnel_cloudflared.homelab.id
      TunnelSecret = random_id.tunnel_secret.b64_std
    })
  }
}

resource "cloudflare_api_token" "external_dns" {
  name = "homelab_external_dns"
  policies = [
    {
      permission_groups = [
        { id = "c8fed203ed3043cba015a93ad1616f1f" }, # Zone:Zone:Read
        { id = "4755a26eedb94da69e1066d98aa820be" }  # Zone:DNS:Edit
      ]
      resources = {
        "com.cloudflare.api.account.zone.*" = "*"
      }
      effect = "allow"
    }
  ]
}

resource "kubernetes_secret" "external_dns_token" {
  metadata {
    name      = "cloudflare-api-token"
    namespace = "external-dns"

    annotations = {
      "app.kubernetes.io/managed-by" = "Terraform"
    }
  }

  data = {
    "value" = cloudflare_api_token.external_dns.value
  }
}

resource "cloudflare_api_token" "cert_manager" {
  name = "homelab_cert_manager"

  policies = [
    {
      permission_groups = [
        { id = "c8fed203ed3043cba015a93ad1616f1f" }, # Zone:Zone:Read
        { id = "4755a26eedb94da69e1066d98aa820be" }  # Zone:DNS:Edit
      ]
      resources = {
        "com.cloudflare.api.account.zone.*" = "*"
      }
      effect = "allow"
    }
  ]
}

resource "kubernetes_secret" "cert_manager_token" {
  metadata {
    name      = "cloudflare-api-token"
    namespace = "cert-manager"

    annotations = {
      "app.kubernetes.io/managed-by" = "Terraform"
    }
  }

  data = {
    "api-token" = cloudflare_api_token.cert_manager.value
  }
}

resource "cloudflare_api_token" "cf_switch" {
  name = "homelab_cf_switch"

  policies = [
    {
      permission_groups = [
        { id = "c8fed203ed3043cba015a93ad1616f1f" }, # Zone:Zone:Read
        { id = "3030687196b94b638145a3953da2b699" }  # Zone:Zone Settings:Write
      ]
      resources = {
        "com.cloudflare.api.account.zone.*" = "*"
      }
      effect = "allow"
    }
  ]
}

resource "kubernetes_secret" "cf_switch_token" {
  metadata {
    name      = "cloudflare-api-token"
    namespace = "cf-switch"

    annotations = {
      "app.kubernetes.io/managed-by" = "Terraform"
    }
  }

  data = {
    "token" = cloudflare_api_token.cf_switch.value
  }
}
