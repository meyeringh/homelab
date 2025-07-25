# Proton

## Manual Action required
1. To initialize exec into container.
2. Kill all proton processes
3. Execute `entrypoint.sh init`
4. Follow: https://hub.docker.com/r/shenxn/protonmail-bridge#initialization
5. Create `/root/cert.pem` and `/root/key.pem` with contents from secret proton/mail-tls-certificate
6. In proton console: `cert import` and follow instructions
7. Kill container. It should start normally afterwards
