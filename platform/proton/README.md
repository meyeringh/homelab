# Proton Mail Bridge

Runs [VideoCurio/ProtonMailBridgeDocker](https://github.com/VideoCurio/ProtonMailBridgeDocker)
(`ghcr.io/videocurio/proton-mail-bridge`), a headless Proton Mail Bridge CLI build.
The bridge listens on `127.0.0.1:1025/1143` inside the container; socat forwards
container ports `25` (SMTP) and `143` (IMAP) to it.

SMTP is only exposed in-cluster: `proton.proton.svc.cluster.local:1025`
(plain SMTP + LOGIN auth, bridge cert is self-signed so no STARTTLS).
Consumers: nextcloud, vaultwarden.

All bridge state (GPG key, pass store, bridge vault with the account login and
the generated SMTP credentials) lives on the retained `state` PVC mounted at `/root`.

## Initialization / re-login (manual)

1. Exec into the container:
   ```
   kubectl --kubeconfig metal/kubeconfig.yaml -n proton exec -it deploy/proton -- bash
   ```
2. `pkill bridge` — the container keeps running (the entrypoint blocks on a FIFO).
3. Start an interactive bridge CLI: `/usr/bin/bridge --cli`
4. `login` — Proton account credentials + 2FA.
5. `info` — note the bridge-local SMTP username and password.
6. `exit` the CLI, leave the pod, then restart it so the entrypoint-managed
   bridge takes over:
   ```
   kubectl --kubeconfig metal/kubeconfig.yaml -n proton delete pod -l app.kubernetes.io/name=proton
   ```
7. Create/update the secret other services consume, using the `info` output:
   ```
   kubectl --kubeconfig metal/kubeconfig.yaml -n proton create secret generic smtp-config-secret \
     --from-literal=username='bridge_smtp_username' \
     --from-literal=password='bridge_smtp_password' \
     --from-literal=sender='sender_name_to_send_mail_from' \
     --from-literal=sender_mail='sender_mail_to_send_mail_from' \
     --from-literal=domain='sender_domain_to_send_mail_from' \
     --dry-run=client -o yaml | kubectl --kubeconfig metal/kubeconfig.yaml apply -f -
   ```

## Credential propagation

`smtp-config-secret` (namespace `proton`) is read by the ClusterSecretStore `proton`
and copied by ExternalSecrets into `nextcloud-mail-secret` (namespace `nextcloud`)
and `vaultwarden-mail-secret` (namespace `vaultwarden`). After updating the secret:

```
kubectl -n nextcloud annotate externalsecret nextcloud-mail-secret force-sync=$(date +%s) --overwrite
kubectl -n vaultwarden annotate externalsecret vaultwarden-mail-secret force-sync=$(date +%s) --overwrite
kubectl -n nextcloud rollout restart deploy/nextcloud
kubectl -n vaultwarden rollout restart deploy/vaultwarden
```

(Consumers read the secret via env vars, so a restart is required.)

## Migration note (shenxn/protonmail-bridge → VideoCurio)

Both images keep their state under `/root` with the same layout (pass store +
bridge vault), so the account login and generated SMTP credentials normally
survive the image switch. After the rollout, verify with `info` in the bridge
CLI; only if the bridge asks for a login again, run the initialization steps
above and update `smtp-config-secret`.
