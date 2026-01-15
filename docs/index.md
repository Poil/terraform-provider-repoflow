---
# Generated from templates/index.md.tmpl
# DO NOT EDIT DIRECTLY

page_title: "Repoflow Terraform Provider"
description: |-
  This is a terraform provider use to manage [RepoFlow](https://www.repoflow.io/) with stable Api
---

# Repoflow Terraform Provider

Manage your [RepoFlow](https://www.repoflow.io/) with terraform.

This provider utilizes the stable api (v1) to interact with Repoflow.

## Setup

### Authentication

```terraform
provider "repoflow" {
  base_url = "https://repoflow.example/api"
  api_key  = "pat_xxx"
}
```

It's best practice not to store the authentication token in plain text. As an alternative, the provider can source the authentication token from the `REPOFLOW_API_KEY` environment variable. It's also possible to use `REPOFLOW_BASE_URL` to define the base url (default is `https://127.0.0.1/api`).
