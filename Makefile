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
	./scripts/backup --action setup --namespace=jellyfin --pvc=jellyfin-data
	./scripts/backup --action setup --namespace=vaultwarden --pvc=vaultwarden-data-vaultwarden-0
	./scripts/backup --action setup --namespace=baikal --pvc=baikal-data
	./scripts/backup --action setup --namespace=seafile --pvc=data-seafile-mariadb-0
	./scripts/backup --action setup --namespace=seafile --pvc=seafile-data
	./scripts/backup --action setup --namespace=joplin --pvc=data-joplin-postgresql-0
	./scripts/backup --action setup --namespace=paperless --pvc=paperless-data
	./scripts/backup --action setup --namespace=webtrees --pvc=data-webtrees-mariadb-0
	./scripts/backup --action setup --namespace=webtrees --pvc=webtrees
	./scripts/backup --action setup --namespace=proton --pvc=proton

restore:
	./scripts/backup --action restore --namespace=actualbudget --pvc=actualbudget
	./scripts/backup --action restore --namespace=jellyfin --pvc=jellyfin-data
	./scripts/backup --action restore --namespace=vaultwarden --pvc=vaultwarden-data-vaultwarden-0
	./scripts/backup --action restore --namespace=baikal --pvc=baikal-data
	./scripts/backup --action restore --namespace=seafile --pvc=data-seafile-mariadb-0
	./scripts/backup --action restore --namespace=seafile --pvc=seafile-data
	./scripts/backup --action restore --namespace=joplin --pvc=data-joplin-postgresql-0
	./scripts/backup --action restore --namespace=paperless --pvc=paperless-data
	./scripts/backup --action restore --namespace=webtrees --pvc=data-webtrees-mariadb-0
	./scripts/backup --action restore --namespace=webtrees --pvc=webtrees
	./scripts/backup --action restore --namespace=proton --pvc=proton

test:
	make -C test

clean:
	docker compose --project-directory ./metal/roles/pxe_server/files down

git-hooks:
	pre-commit install
