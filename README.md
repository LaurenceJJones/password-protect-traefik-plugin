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

```yaml
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
        - password-protect1

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

### Labels

You can instead define labels on the containers themselves if you want to not use a static or dynamic configuration:

```yaml
services:
  traefik:
    image: "traefik:v3.0.0"
    container_name: "traefik"
    restart: unless-stopped
    command:
      - "--api.insecure=true"
      - "--providers.docker=true"
      - "--providers.docker.exposedbydefault=false"
      - "--entrypoints.web.address=:80"

      - "--experimental.plugins.password-protect.modulename=github.com/LaurenceJJones/password-protect-traefik-plugin"
      - "--experimental.plugins.password-protect.version=v0.0.3"
    volumes:
      - "/var/run/docker.sock:/var/run/docker.sock:ro"
      - './login.html:/login.html:ro'
```

Then on your service you can define label only configuration:

```yaml
  whoami-foo:
    image: traefik/whoami
    container_name: "simple-service-foo"
    restart: unless-stopped
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.router-foo.rule=PathPrefix(`/foo`)"
      - "traefik.http.routers.router-foo.entrypoints=web"
      - "traefik.http.services.service-foo.loadbalancer.server.port=80"
      - "traefik.http.routers.router-foo.middlewares=password-protect1@docker"
      - "traefik.http.middlewares.password-protect1.plugin.password-protect.password=Y0uRSup3RP@ssW0rD"
```

In the example we define a `http.middlewares` called `password-protect1` that calls the `plugin.password-protect` and sets `password` to a super value. Then to apply this middleware to the `router-foo` as we define `middlewares` is `password-protect1@docker`.

## Flow

So how does this plugin work?

When it recieves a http request it checks for the cookie named "spp-session" (simple password protect session), the cookie is a UUID and a signature format EG: "uuid.signature" the signature is the signed value of the UUID using the password as the signer value. This means that if the cookie is tampered or modified in any way then the signature check will fail and they will be prompted again for the password.

Once the user has entered the valid password for the middleware, they will be sent a redirect request including the set-cookie attribute which as stated before is the uuid and signature value. The cookie is only valid for the user session, as soon as they close their browser the session will be lost and they need to reauthenticate with the password upon coming back.

The only security concern that we have is we do not track UUID's at all, so if the cookie is stolen there is no way to invalidate a session, so in this case the best course of action would be to set another password for this middleware and then all signature checks will fail and they will be prompted again for the new password.

## Security

This middleware should **NEVER** replace your own authentication mechanisms on your application, its simply a way to provide a first barrier to bots/crawlers that might be tracking newly created TLS/SSL certificates via https://crt.sh as Security is in layers. As explained in the "why" section we enjoyed the Vercel deployment password, because we might be updating a public blog post or something that isnt behind the application authentication and didn't want it to be truly publically accessible (Yes we could IP allow/deny but again we found this was causing headaches to maintain and a simply password would do).
