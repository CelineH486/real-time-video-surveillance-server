# Production deployment

This guide prepares the application for a Linux server with Docker Compose,
NGINX, a DNS name, and a TLS certificate.

## 1. DNS and environment

Point both production DNS names to the server's public IP:

- `monitor.example.com` for the dashboard and API.
- `stream.monitor.example.com` for MediaMTX WHEP signaling.

Create the production environment file:

```bash
cp .env.production.example .env.production
chmod 600 .env.production
```

Replace every `change-me` value and both example DNS names.
Generate independent random values for the database password, stream signing
key, and publisher password. Do not commit `.env.production`.

`MEDIAMTX_WEBRTC_ADDITIONAL_HOSTS` must contain the public DNS name or IP that
the browser can use for WebRTC ICE connections.

## 2. Start the application

Validate the merged Compose configuration before starting:

```bash
docker compose --env-file .env.production \
  -f compose.yaml -f compose.prod.yaml config --quiet

docker compose --env-file .env.production \
  -f compose.yaml -f compose.prod.yaml up -d --build
```

Check the API locally on the server:

```bash
curl http://127.0.0.1:8080/health
docker compose --env-file .env.production \
  -f compose.yaml -f compose.prod.yaml ps
```

The production Compose file intentionally binds ports `8080`, `8889`, and
`9996` to `127.0.0.1`. They are reached through NGINX and must not be opened
directly to the internet.

## 3. NGINX and TLS

Install NGINX and Certbot using the server distribution's package manager.
Obtain a certificate only after DNS points to the server and inbound TCP ports
80 and 443 are open.

Copy `deploy/nginx/surveillance.conf.example` to the NGINX sites directory,
replace both example DNS names, and enable the site. The certificate paths in
the example use the standard Let's Encrypt layout.

Example certificate request:

```bash
sudo certbot --nginx \
  -d monitor.example.com \
  -d stream.monitor.example.com
sudo nginx -t
sudo systemctl reload nginx
```

After deployment, open:

```text
https://monitor.example.com/web/
```

## 4. Firewall

Allow only the ports required by the deployment:

| Port | Protocol | Purpose | Recommended source |
|---|---|---|---|
| 80 | TCP | ACME and HTTPS redirect | Internet |
| 443 | TCP | HTTPS dashboard, API, WHEP signaling | Internet |
| 8189 | TCP/UDP | WebRTC ICE media | Internet |
| 8554 | TCP | Truck RTSP publishing | Truck networks only |
| 5000 | UDP | Truck heartbeat | Truck networks only |
| 22 | TCP | SSH administration | Administrator IPs only |

Do not expose PostgreSQL, `8080`, `8889`, `9996`, or `9997` publicly.
Configure both the cloud-provider firewall and the server firewall.

## 5. Verification

Verify these items from a device outside the server network:

1. `https://<domain>/health` returns a successful response.
2. `https://<domain>/login` opens without a certificate warning.
3. Login and logout work.
4. An authorized truck and its cameras are visible.
5. WHEP requests use `https://stream.<domain>/...`.
6. Live video plays through WebRTC on desktop and mobile.
7. Ports `8080`, `8889`, `9996`, `9997`, and PostgreSQL are unreachable from
   the public internet.
