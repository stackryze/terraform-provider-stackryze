terraform {
  required_providers {
    stackryze = {
      source = "stackryze/stackryze"
    }
  }
}

provider "stackryze" {
  # api_token can also come from STACKRYZE_API_TOKEN
  # api_token = "sk_dns_xxx"
}

# Look up an existing zone by name.
data "stackryze_zone" "example" {
  name = "example.com"
}

# Apex A record.
resource "stackryze_record" "root" {
  zone_id = data.stackryze_zone.example.id
  name    = "@"
  type    = "A"
  content = "93.184.216.34"
  ttl     = 3600
}

# www CNAME.
resource "stackryze_record" "www" {
  zone_id = data.stackryze_zone.example.id
  name    = "www"
  type    = "CNAME"
  content = "example.com"
}

# MX (priority is part of content).
resource "stackryze_record" "mail" {
  zone_id = data.stackryze_zone.example.id
  name    = "@"
  type    = "MX"
  content = "10 mail.example.com"
}
