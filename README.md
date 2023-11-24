# HTTPS reverse proxy for Tailscale

Expose web services to your tailnet as separate devices, with TLS.

## Configuration

The following configuration options are available, passed as environment variables:

- `BACKEND`: the backend service url, default: `http://127.0.0.1:8080`.
- `HOSTNAME`: the tailnet hostname for the service.
- `PORT`: the port the service listens on, default: `443`.
- `STATE_DIR`: the state store directory, default: `/var/lib/tsrp`.
- `TS_AUTHKEY`: the auth key to create the node in Tailscale.
- `VERBOSE`: show tailscale logs, default: `false`.

## Usage

[Enable HTTPS](https://tailscale.com/kb/1153/enabling-https/) and [create an auth key](https://tailscale.com/kb/1085/auth-keys/) in Tailscale.

The proxy can be used in docker compose as a sidecar to any HTTP service. For an example check the [docker-compose.yml](docker-compose.yml).
