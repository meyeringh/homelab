.POSIX:

default: apply

~/.terraform.d/credentials.tfrc.json:
	tofu login app.terraform.io

terraform.tfvars:
	cp terraform.tfvars.example ${@}
	nvim ${@}

.terraform.lock.hcl: ~/.terraform.d/credentials.tfrc.json versions.tf terraform.tfvars
	tofu init
	touch ${@}

namespaces:
	ansible-playbook namespaces.yml

plan: .terraform.lock.hcl
	KUBE_CONFIG_PATH=../metal/kubeconfig.yaml tofu plan

apply: .terraform.lock.hcl namespaces
	KUBE_CONFIG_PATH=../metal/kubeconfig.yaml tofu apply -auto-approve
