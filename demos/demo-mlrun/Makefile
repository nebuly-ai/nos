HOST_IP ?= 127.0.0.1
MLRUN_DATA_DIR ?= ~/mlrun-data

.PHONY: deploy
install:
	HOST_IP=$(HOST_IP) SHARED_DIR=$(MLRUN_DATA_DIR) docker-compose up

clean:
	HOST_IP=$(HOST_IP) SHARED_DIR=$(MLRUN_DATA_DIR) docker-compose down
