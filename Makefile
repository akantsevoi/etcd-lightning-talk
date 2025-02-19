.PHONY: up
up:
	docker-compose -f etcd/docker-compose.yaml up -d

.PHONY: down
down:
	docker-compose -f etcd/docker-compose.yaml down

.PHONY: install-tc
install-tc:
	for node in $$(docker ps --filter "name=etcd-etcd-0*" --format "{{.Names}}"); do \
		echo "Install tc for $$node"; \
		docker exec $$node sh -c 'apt-get update && apt-get install -y iproute2'; \
	done

.PHONY: add-delays
DELAY ?= 200
NETWORK_NAME := etcd
add-delays:
	for node in $$(docker ps --filter "name=etcd-etcd-*" --format "{{.Names}}"); do \
		echo "Adding delay to $$node"; \
		docker exec "$$node" tc qdisc replace dev eth0 root netem delay ${DELAY}ms; \
	done

.PHONY: remove-delays
remove-delays:
	for node in $$(docker ps --filter "name=etcd-etcd-0*" --format "{{.Names}}"); do \
		echo "Removing delay from $$node"; \
		docker exec "$$node" tc qdisc del dev eth0 root || true; \
	done