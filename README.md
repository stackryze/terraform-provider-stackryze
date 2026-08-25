# Terraform Provider for Stackryze DNS

Manage DNS records on [Stackryze DNS](https://dns.stackryze.com) as code. The
provider is a thin client over the Stackryze REST API using an API token — no
impact on the DNS serving path.

## Usage

```hcl
terraform {
  required_providers {
    stackryze = {
      source = "stackryze/stackryze"
    }
  }
}

provider "stackryze" {
  # api_token = "sk_dns_xxx"   # or STACKRYZE_API_TOKEN
  # api_url   = "https://api.stackryze.com/api"  # or STACKRYZE_API_URL
}

data "stackryze_zone" "example" {
  name = "example.com"
}

resource "stackryze_record" "www" {
  zone_id = data.stackryze_zone.example.id
  name    = "www"
  type    = "CNAME"
  content = "example.com"
  ttl     = 3600
}
```

## Authentication

Create an API token with **write** scope in Settings → API tokens, then set:

```bash
export STACKRYZE_API_TOKEN=sk_dns_xxxxxxxx
```

## Resources & data sources

| Name | Kind | Purpose |
|------|------|---------|
| `stackryze_zone` | data source | Resolve a zone id by name (zones are created in the app first) |
| `stackryze_record` | resource | Manage a single DNS record |

### `stackryze_record`

| Attribute | Required | Notes |
|-----------|----------|-------|
| `zone_id` | yes | From the `stackryze_zone` data source. Changing forces replacement. |
| `name` | yes | Label relative to the zone; `@` for apex. Forces replacement. |
| `type` | yes | A, AAAA, CNAME, MX, TXT, SRV, CAA. Forces replacement. |
| `content` | yes | Value. For MX/SRV include priority (`10 mail.example.com`). Forces replacement. |
| `ttl` | no | Seconds, min 3600 (default). |

Import an existing record:

```bash
terraform import stackryze_record.www "ZONE_ID/CNAME/www/example.com"
```

## Build (local)

Requires Go 1.22+.

```bash
go mod tidy
go build -o terraform-provider-stackryze
```

For local testing, add a dev override to `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "stackryze/stackryze" = "/absolute/path/to/this/repo"
  }
  direct {}
}
```

Then run `terraform plan` from the `examples/` directory.

## Notes

- Zones must already exist on Stackryze (creation requires nameserver / ownership
  verification, which is done in the dashboard). This provider manages records.
- Record identity is `zone_id + type + name + content`; changing any of these
  replaces the record.
