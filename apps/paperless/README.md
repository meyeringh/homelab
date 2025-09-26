# Paperless

## Manual Action required
1. Create app password in nextcloud for paperless
2. Create secret for rclone container, to sync files from nextcloud to:
    ```
    kubectl --kubeconfig metal/kubeconfig.yaml create secret generic nextcloud-webdav-secret --namespace=paperless \
    --from-literal=rclone.conf="$(cat <<'EOF'
    [nextcloud]
    type = webdav
    url = https://YOUR-NEXTCLOUD/remote.php/dav/files/YOUR-USER/Scans
    vendor = nextcloud
    user = YOUR-USER
    pass = YOUR-NEXTCLOUD-APP-PASSWORD
    EOF
    )"
    ```