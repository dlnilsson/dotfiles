export VIRTUALBOX_DISK_SIZE=40000

composer_install() {
	docker run --rm -ti \
	-v $(pwd):/tmp/source \
	--workdir '/tmp/source' \
	-v $COMPOSER_CACHE_DIR:/var/run/composer/cache \
	$AWS_ACC/php-utilities:latest \
	bash -c "composer install --working-dir=/tmp/source --ignore-platform-reqs --no-suggest"
}
c0mposer() {
	docker run --rm -ti \
	-v $(pwd):/tmp/source \
	--workdir '/tmp/source' \
	-v $COMPOSER_CACHE_DIR:/var/run/composer/cache \
	$AWS_ACC/php-utilities:latest \
	composer $*
}
php_security_checker() {
	docker run --rm -ti \
	-v $(pwd):/tmp \
	$AWS_ACC/php-utilities:latest \
	security-checker security:check /tmp/composer.lock
}

inteleon() {
	ssh -o StrictHostKeyChecking=no -i $INT_PEM $INT_USR@$*
}
inteleon_ssh() {
	vault ssh -strict-host-key-checking=no -address=$VAULT_PROD_URL -role $INT_USR -mode otp ubuntu@$*
}
aidssh() {
	inteleon_ssh $(awless ls instances --filter id=$1 | awk 'FNR==3 {print $13}')
}
swarm_prod() {
	docker --tls -H $INT_SWARM_PRODUCTION $*
}
swarm_staging() {
	docker --tls -H $INT_SWARM_STAGING $*
}
eval_prod() {
	eval_local
	export DOCKER_TLS=1
	export DOCKER_HOST=$INT_SWARM_PRODUCTION
}
eval_staging() {
	eval_local
	export DOCKER_TLS=1
	export DOCKER_HOST=$INT_SWARM_STAGING
}
eval_manager() {
	eval $(docker-machine env virtualbox-node-manager-1)
}
eval_local() {
	# https://docs.docker.com/engine/reference/commandline/cli/#environment-variables
	local variables=(
		DOCKER_API_VERSION
		DOCKER_CONFIG
		DOCKER_CERT_PATH
		DOCKER_DRIVER
		DOCKER_HOST
		DOCKER_NOWARN_KERNEL_VERSION
		DOCKER_RAMDISK
		DOCKER_TLS
		DOCKER_TLS_VERIFY
		DOCKER_CONTENT_TRUST
		DOCKER_CONTENT_TRUST_SERVER
		DOCKER_HIDE_LEGACY_COMMANDS
		DOCKER_TMPDIR
		DOCKER_MACHINE_NAME
	)
	for i in $variables; do
		unset $i
	done
}
eval_worker() {
	eval $(docker-machine env virtualbox-node-worker-1)
}
add_ssh_key_local_swarm() {
	eval_worker
	cat $PUBLIC_SSH_KEY | docker exec -i $(docker ps --filter='name=ssh' -q) /bin/bash -c "cat >> /root/.ssh/authorized_keys"
	info_msg "Added $PUBLIC_SSH_KEY to ssh container"
}
