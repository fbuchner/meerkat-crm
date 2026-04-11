---
title: FAQ & Troubleshooting
nav_order: 9
has_children: false
---

# FAQ & Troubelshooting

1. When registering a new user or signing in I get the error `Failed to execute 'json' on 'Response': Unexpected end of JSON input`.

**Answer**: Make sure to set the environment variable `FRONTEND_URL` in your .env.docker to the correct URL.


1. When logging in it works but then redirects to login again.

**Answer**: Make sure to change the `COOKIE_DOMAIN` environment variable in your .env.docker to your host IP.

1. When using an OIDC login provider the user is not found. 

**Answer**: The OIDC provider has to either set the property email_verified=true or you have to set the OIDC_TRUST_EMAIL=true environment variable.

