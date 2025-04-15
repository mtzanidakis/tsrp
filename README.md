# HTTPS reverse proxy for Tailscale

Expose web services to your tailnet as separate devices, with TLS.

## Configuration

The following configuration options are available, passed as environment variables:

- `BACKEND`: the backend service url, default: `http://127.0.0.1:8080`.
- `FUNNEL`: expose service to the public internet with Funnel on HTTPS_PORT (requires port 443, 8443 or 10000), default: `false`.
- `HOSTNAME`: the tailnet hostname for the service.
- `HTTP_PORT`: the http port the service listens on for redirecting to https, default: `80`.
- `HTTPS_PORT`: the port the service listens on, default: `443`.
- `STATE_DIR`: the state store directory, default: `/var/lib/tsrp`.
- `TS_AUTHKEY`: the auth key to create the node in Tailscale.
- `VERBOSE`: show tailscale logs, default: `false`.

## Usage

[Enable HTTPS](https://tailscale.com/kb/1153/enabling-https/) and [create an auth key](https://tailscale.com/kb/1085/auth-keys/) in Tailscale.

The proxy can be used in docker compose as a sidecar to any HTTP service. For an example check the [docker-compose.yml](docker-compose.yml).

## Funnel

Funnel is a Tailscale feature that allows you to expose your services to the public internet, alongside your tailnet. To use it, you need to set the `FUNNEL` environment variable to `true` and ensure that the `HTTPS_PORT` is set to one of the following ports: `443`, `8443`, or `10000`. Tailscale Funnel requires a node attribute (nodeAttrs) of funnel in your tailnet policy file to tell Tailscale who can use Funnel, follow the [instructions](https://tailscale.com/kb/1223/funnel#funnel-node-attribute). The proxy will then automatically configure Funnel for you.
