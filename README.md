# DDNS allowlist - Traefik plugin

Dynamic DNS allowlist plugin for Traefik: Add your dynamic hostname (your homenetwork router) to the allow list

## About

The `ddns-allowlist` plugin for Traefik allows you to add dynamic DNS (DDNS) hosts to the allowed requesters.
Requests from IP addresses that do not resolve to the specified DDNS hosts will be denied.

This idea was created to add your router with a floating ips to an allowlist.
This is not limited to your DDNS supporting router - you can add any host.
It is more an hostname allowlist which will do a DNS lookup.
Because server typically have a static IP, you should add its static IPs to the allowlist.

The existing plugins can be browsed into the [Plugin Catalog](https://plugins.traefik.io/plugins/66fbe453573cd7803d65cb10/ddns-allowlist).

## Installation

To install the `ddns-allowlist` plugin, add the following configuration to your Traefik static configuration:

```yaml
experimental:
  plugins:
    ddns-allowlist:
      moduleName: "github.com/taskmedia/ddns-allowlist"
      version: v1.4.0
```

## Configuration

Add the `ddns-allowlist` middleware to your Traefik dynamic configuration:

```yaml
# Dynamic configuration

http:
  routers:
    my-router:
      rule: host(`demo.localhost`)
      service: service-foo
      entryPoints:
        - web
      middlewares:
        - ddns-allowlist-router

  services:
    service-foo:
      loadBalancer:
        servers:
          - url: http://127.0.0.1:5000

  middlewares:
    ddns-allowlist-router:
      plugin:
        ddns-allowlist:
          logLevel: ERROR
          hostList: # hosts to dynamically allowlist via DNS lookup
            - my.router.ddns.tld
          ipList: # optional IP addresses to allowlist
            - 1.2.3.4
```
