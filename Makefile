.POSIX:
.PHONY: *
.EXPORT_ALL_VARIABLES:

KUBECONFIG = $(shell pwd)/metal/kubeconfig.yaml
KUBE_CONFIG_PATH = $(KUBECONFIG)

default: metal system external smoke-test post-install clean

configure:
	./scripts/configure
	git status

metal:
	make -C metal

system:
	make -C system

external:
	make -C external

smoke-test:
	make -C test filter=Smoke

post-install:
	@./scripts/hacks

# TODO maybe there's a better way to manage backup with GitOps?
backup:
	./scripts/backup --action setup --namespace=actualbudget --pvc=actualbudget
	./scripts/backup --action setup --namespace=jellyfin --pvc=jellyfin
	./scripts/backup --action setup --namespace=vaultwarden --pvc=vaultwarden-data-vaultwarden-0
	./scripts/backup --action setup --namespace=joplin --pvc=data-joplin-postgresql-0
	./scripts/backup --action setup --namespace=paperless --pvc=paperless
	./scripts/backup --action setup --namespace=webtrees --pvc=data-webtrees-mariadb-0
	./scripts/backup --action setup --namespace=webtrees --pvc=webtrees
	./scripts/backup --action setup --namespace=proton --pvc=proton
	./scripts/backup --action setup --namespace=minecraft --pvc=minecraft
	./scripts/backup --action setup --namespace=home --pvc=home-home-assistant-home-home-assistant-0
	./scripts/backup --action setup --namespace=nextcloud --pvc=data-nextcloud-postgresql-0
	./scripts/backup --action setup --namespace=nextcloud --pvc=nextcloud-nextcloud
	./scripts/backup --action setup --namespace=nextcloud --pvc=redis-data-nextcloud-redis-master-0

restore:
	./scripts/backup --action restore --namespace=actualbudget --pvc=actualbudget
	./scripts/backup --action restore --namespace=jellyfin --pvc=jellyfin
	./scripts/backup --action restore --namespace=vaultwarden --pvc=vaultwarden-data-vaultwarden-0
	./scripts/backup --action restore --namespace=joplin --pvc=data-joplin-postgresql-0
	./scripts/backup --action restore --namespace=paperless --pvc=paperless
	./scripts/backup --action restore --namespace=webtrees --pvc=data-webtrees-mariadb-0
	./scripts/backup --action restore --namespace=webtrees --pvc=webtrees
	./scripts/backup --action restore --namespace=proton --pvc=proton
	./scripts/backup --action restore --namespace=minecraft --pvc=minecraft
	./scripts/backup --action restore --namespace=home --pvc=home-home-assistant-home-home-assistant-0
	./scripts/backup --action restore --namespace=nextcloud --pvc=data-nextcloud-postgresql-0
	./scripts/backup --action restore --namespace=nextcloud --pvc=nextcloud-nextcloud
	./scripts/backup --action restore --namespace=nextcloud --pvc=redis-data-nextcloud-redis-master-0


test:
	make -C test

clean:
	docker compose --project-directory ./metal/roles/pxe_server/files down

git-hooks:
	pre-commit install
