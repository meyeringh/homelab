data "cloudflare_zones" "zone" {
  name = "meyeringh.org"
}

# meyeringh.org is a static site on GitHub Pages (meyeringh/meyeringh-org)
resource "cloudflare_dns_record" "pages_apex_a" {
  for_each = toset([
    "185.199.108.153",
    "185.199.109.153",
    "185.199.110.153",
    "185.199.111.153",
  ])

  zone_id = data.cloudflare_zones.zone.result[0].id
  type    = "A"
  name    = "meyeringh.org"
  content = each.value
  proxied = false
  ttl     = 1 # Auto
}

resource "cloudflare_dns_record" "pages_apex_aaaa" {
  for_each = toset([
    "2606:50c0:8000::153",
    "2606:50c0:8001::153",
    "2606:50c0:8002::153",
    "2606:50c0:8003::153",
  ])

  zone_id = data.cloudflare_zones.zone.result[0].id
  type    = "AAAA"
  name    = "meyeringh.org"
  content = each.value
  proxied = false
  ttl     = 1 # Auto
}

resource "cloudflare_dns_record" "pages_www" {
  zone_id = data.cloudflare_zones.zone.result[0].id
  type    = "CNAME"
  name    = "www"
  content = "meyeringh.github.io"
  proxied = false
  ttl     = 1 # Auto
}

resource "cloudflare_api_token" "external_dns" {
  name = "homelab_external_dns"
  policies = [
    {
      permission_groups = [
        { id = "c8fed203ed3043cba015a93ad1616f1f" }, # Zone:Zone:Read
        { id = "4755a26eedb94da69e1066d98aa820be" }  # Zone:DNS:Edit
      ]
      resources = jsonencode({ "com.cloudflare.api.account.zone.*" = "*" })
      effect    = "allow"
    }
  ]
}

resource "kubernetes_secret_v1" "external_dns_token" {
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
      resources = jsonencode({ "com.cloudflare.api.account.zone.*" = "*" })
      effect    = "allow"
    }
  ]
}

resource "kubernetes_secret_v1" "cert_manager_token" {
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

resource "cloudflare_api_token" "cloudflare_ddns" {
  name = "homelab_cloudflare_ddns"

  policies = [
    {
      permission_groups = [
        { id = "c8fed203ed3043cba015a93ad1616f1f" }, # Zone:Zone:Read
        { id = "4755a26eedb94da69e1066d98aa820be" }  # Zone:DNS:Edit
      ]
      resources = jsonencode({ "com.cloudflare.api.account.zone.*" = "*" })
      effect    = "allow"
    }
  ]
}

resource "kubernetes_secret_v1" "cloudflare_ddns_token" {
  metadata {
    name      = "cloudflare-api-token"
    namespace = "cloudflare-ddns"

    annotations = {
      "app.kubernetes.io/managed-by" = "Terraform"
    }
  }

  data = {
    "token" = cloudflare_api_token.cloudflare_ddns.value
  }
}
