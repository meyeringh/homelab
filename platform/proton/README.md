# Proton

## Manual Action required
1. To initialize exec into container.
2. Kill all proton processes
3. Execute `entrypoint.sh init`
4. Follow: https://hub.docker.com/r/shenxn/protonmail-bridge#initialization
5. Create secret for other services to use, with output from `info` command:
    ```
    kubectl --kubeconfig metal/kubeconfig.yaml create secret generic smtp-config-secret --namespace=proton \
    --from-literal=username='username_login_for_proton_bridge' \
    --from-literal=password='password_login_for_proton_bridge' \
    --from-literal=sender='sender_name_to_send_mail_from' \
    --from-literal=sender_mail='sender_mail_to_send_mail_from'
    ```
6. Create `/root/cert.pem` and `/root/key.pem` with contents from secret proton/mail-tls-certificate
7. In proton console: `cert import` and follow instructions
8. Kill container. It should start normally afterwards
