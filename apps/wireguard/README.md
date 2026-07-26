# WireGuard

Road-warrior VPN for the phone. Each phone tunnel has two peers: one
ProtonVPN server (default route, internet privacy) and this server
(access to `192.168.1.0/24`). Switching Proton servers = switching
tunnels in the WireGuard Android app. Only one Proton connection is
active at a time, and internet traffic goes phone → Proton directly
(no hairpin through the home connection).

## Setup

1. Generate the server key and store it via OpenTofu:

   ```bash
   wg genkey   # add as extra_secrets.wireguard-private-key in external/terraform.tfvars
   make external
   ```

2. Router: forward UDP `51820` to `192.168.1.6` (the pinned LB IP).
   `wg.meyeringh.org` is kept on the WAN IP by cloudflare-ddns.

3. In the [ProtonVPN dashboard](https://account.protonvpn.com/downloads)
   generate one WireGuard config per server you want as an option and
   download them into a directory (not into the repo — they contain a
   private key).

4. Build the phone tunnels and print the phone's public key:

   ```bash
   scripts/wireguard-phone-config ~/proton-confs --qr
   ```

   Put the printed `publicKey`/`allowedIPs` into `peers` in
   `values.yaml`, push, and wait for ArgoCD to sync. Then scan the QR
   codes in the WireGuard app (one tunnel per Proton server) and delete
   the generated configs.

5. Verify:

   ```bash
   KUBECONFIG=metal/kubeconfig.yaml kubectl -n wireguard exec deploy/wireguard -- wg show
   ```

## Notes

- The server's `wg0.conf` is rendered by an ExternalSecret; after
  changing keys or peers, restart the deployment (subPath mounts do not
  update in place).
- Set the WireGuard app as always-on VPN in Android settings.
- Using the tunnels from inside the home network requires NAT hairpin
  on the router (works like rustdesk); otherwise toggle the tunnel off
  at home.
- All tunnels share the private key of one Proton config (Proton
  registers keys account-wide), so the server needs only one phone peer.
