# Password Protect Traefik Plugin

A Traefik middleware plugin that implements simple password protection with cookie-based sessions.

## Why develop this?

We migrated our deployments away from Vercel and still wanted to keep password protected developer deployments for clients as using basic authentication or LDAP would require external maintence. Since we enjoyed the simplistic approach of generating a random password per client, then we wanted to implement this in traefik as we use Coolify as our new CI/CD tool.

## Features
- Password protection for any Traefik service
- Cookie-based session management
- Customizable login page with light/dark mode
- Easy integration with Traefik middleware system

## Usage

Firstly you should download the `login.html` file under `templates/dist/` as we need this page to be mounted to `/login.html` within the traefik container since we cant compile the `login.html` into the source code:

```
services:
  traefik:
    image: "traefik:v3.0.0"
    container_name: "traefik"
    restart: unless-stopped
    volumes:
      - './login.html:/login.html:ro'
```

Then within your static configuration you can define the `password-protect` plugin.

```yaml
# Static configuration

experimental:
  plugins:
    password-protect:
      moduleName: github.com/LaurenceJJones/password-protect-traefik-plugin
      version: vX.Y.Z # To update
```

Then within your dynamic configuration you can define the middlewares as a name EG: `password-protect1` with the password configured (_yes I know plain text password but its a super simple password protection page_): 

```yaml
# Dynamic configuration

http:
  routers:
    my-router:
      rule: host(`whoami.localhost`)
      service: service-foo
      entryPoints:
        - web
      middlewares:
        - password-protect

  services:
    service-foo:
      loadBalancer:
        servers:
          - url: http://127.0.0.1:5000

  middlewares:
    password-protect1:
      plugin:
        password-protect:
          password: Y0uRSup3RP@ssW0rD
```

## Flow

So how does this plugin work?

When it recieves a http request it checks for the cookie named "spp-session" (simple password protect session), the cookie is a UUID and a signature format EG: "<uuid>.<signature>" the signature is the signed value of the UUID using the password as the signer value. This means that if the cookie is tampered or modified in any way then the signature check will fail and they will be prompted again for the password.

Once the user has entered the valid password for the middleware, they will be sent a redirect request including the set-cookie attribute which as stated before is the uuid and signature value. The cookie is only valid for the user session, as soon as they close their browser the session will be lost and they need to reauthenticate with the password upon coming back.

The only security concern that we have is we do not track UUID's at all, so if the cookie is stolen there is no way to invalidate a session, so in this case the best course of action would be to set another password for this middleware and then all signature checks will fail and they will be prompted again for the new password.
