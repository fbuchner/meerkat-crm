---
title: FAQ & Troubleshooting
nav_order: 9
has_children: false
---

# FAQ & Troubelshooting

### When registering a new user or signing in I get the error `Failed to execute 'json' on 'Response': Unexpected end of JSON input`.

**Answer**: Make sure to set the environment variable `FRONTEND_URL` in your .env to the correct URL.


### When logging in it works but then redirects to login again, or the browser console shows that the `auth_token` cookie was rejected because of an invalid domain.

**Answer**: Leave the `COOKIE_DOMAIN` environment variable in your .env **empty** and restart the backend.

`COOKIE_DOMAIN` is written into the `Domain` attribute of the auth cookie. When it is empty, the cookie becomes a *host-only* cookie that is automatically valid for whatever host you opened Meerkat on - `localhost`, a LAN IP, or a domain. If the value does not match that host, the browser silently discards the cookie and you get bounced back to the login page. An IP address will never work as value. Localhost is rejected unless actually using http://localhost as URL.
calhost` has none.

Only set `COOKIE_DOMAIN` if you serve the frontend and the API from *different subdomains* - for example `app.example.com` and `api.example.com`, where you would use `.example.com`. That setup also needs HTTPS and `COOKIE_SECURE='true'`.

### When using an OIDC login provider the user is not found. 

**Answer**: The OIDC provider has to either set the property email_verified=true or you have to set the OIDC_TRUST_EMAIL=true environment variable.

