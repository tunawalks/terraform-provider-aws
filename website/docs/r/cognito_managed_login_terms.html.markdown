---
subcategory: "Cognito IDP (Identity Provider)"
layout: "aws"
page_title: "AWS: aws_cognito_managed_login_terms"
description: |-
  Manages managed login terms for a Cognito user pool client.
---

# Resource: aws_cognito_managed_login_terms

Manages managed login terms for a Cognito user pool client.

## Example Usage

```terraform
resource "aws_cognito_managed_login_terms" "example" {
  client_id    = aws_cognito_user_pool_client.example.id
  user_pool_id = aws_cognito_user_pool.example.id

  enforcement  = "NONE"
  terms_name   = "terms-of-use"
  terms_source = "LINK"

  links = {
    "cognito:default" = "https://example.com/terms"
    "cognito:spanish" = "https://example.com/terms/es"
  }
}
```

## Argument Reference

The following arguments are required:

* `client_id` - (Required) App client ID the terms apply to. Must be 1-128 characters matching pattern `[\w+]+`.
* `enforcement` - (Required) How acceptance is enforced. Currently only accepts `NONE`.
* `links` - (Required) Map of terms links by language. Must include a `cognito:default` entry. Maximum 12 entries. Valid keys: `cognito:default`, `cognito:english`, `cognito:french`, `cognito:spanish`, `cognito:german`, `cognito:bahasa-indonesia`, `cognito:italian`, `cognito:japanese`, `cognito:korean`, `cognito:portuguese-brazil`, `cognito:chinese-simplified`, `cognito:chinese-traditional`. Values must be 1-1024 characters.
* `terms_name` - (Required) Type of terms document. Must be exactly `terms-of-use` or `privacy-policy`.
* `terms_source` - (Required) Source type for the terms. Currently only accepts `LINK`.
* `user_pool_id` - (Required) User pool ID the client belongs to. Must be 1-55 characters matching pattern `[\w-]+_[0-9a-zA-Z]+`.

The following arguments are optional:

* `region` - (Optional) Region where this resource will be [managed](https://docs.aws.amazon.com/general/latest/gr/rande.html#regional-endpoints). Defaults to the Region set in the [provider configuration](https://registry.terraform.io/providers/hashicorp/aws/latest/docs#aws-configuration-reference).

## Attribute Reference

This resource exports the following attributes in addition to the arguments above:

* `creation_date` - Date and time when the terms were created in RFC3339 format.
* `last_modified_date` - Date and time when the terms were last modified in RFC3339 format.
* `managed_login_terms_id` - UUID v4 identifier of the managed login terms resource.

## Import

In Terraform v1.5.0 and later, use an [`import` block](https://developer.hashicorp.com/terraform/language/import) to import managed login terms using `user_pool_id` and `managed_login_terms_id` separated by `,`. For example:

```terraform
import {
  to = aws_cognito_managed_login_terms.example
  id = "us-west-2_rSss9Zltr,18a6b81e-1b5b-4c9d-82c0-123456789012"
}
```

Using `terraform import`, import managed login terms using `user_pool_id` and `managed_login_terms_id` separated by `,`. For example:

```console
% terraform import aws_cognito_managed_login_terms.example us-west-2_rSss9Zltr,18a6b81e-1b5b-4c9d-82c0-123456789012
```
