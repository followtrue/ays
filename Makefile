GO_CMD=go
GO_BUILD=$(GO_CMD) build
GO_CLEAN=$(GO_CMD) clean
PACK_CMD=tar zcvf

INSTALL_CONFIG_ORG=cmd/installConfig/installConfig.go
MANAGER_ORG=cmd/manager/daemonManager.go
NODE_ORG=cmd/node/daemonNode.go

BIN_DIR=cmd/bin/
INSTALL_CONFIG_FILE=installConfig
MANAGER_FILE=aysManager
NODE_FILE=aysNode
TAR_FILE=ays.tar.gz

ENV_PATH=config/env/env.json
ENV_PRO=env_example/pro.json
ENV_GRAY=env_example/gray.json
ENV_UAT=env_example/uat.json

ENV_USE_PRO=cp $(ENV_PRO) $(ENV_PATH) && $(BIND_DATA) && $(REPLACE_BIND_NAME)
ENV_USE_GRAY=cp $(ENV_GRAY) $(ENV_PATH) && $(BIND_DATA) && $(REPLACE_BIND_NAME)
ENV_USE_UAT=cp $(ENV_UAT) $(ENV_PATH) && $(BIND_DATA) && $(REPLACE_BIND_NAME)

all: clean build

build:
	# build install config
	$(GO_BUILD) -o $(BIN_DIR)$(INSTALL_CONFIG_FILE) -v $(INSTALL_CONFIG_ORG)

	# build daemon manager
	$(GO_BUILD) -o $(BIN_DIR)$(MANAGER_FILE) -v $(MANAGER_ORG)

	# build daemon node
	$(GO_BUILD) -o $(BIN_DIR)$(NODE_FILE) -v $(NODE_ORG)

	# build package
	$(PACK_CMD) $(BIN_DIR)$(TAR_FILE) -C $(BIN_DIR) $(INSTALL_CONFIG_FILE) $(MANAGER_FILE) $(NODE_FILE)

clean:
	# clean build file
	rm -f $(BIN_DIR)$(INSTALL_CONFIG_FILE)
	rm -f $(BIN_DIR)$(MANAGER_FILE)
	rm -f $(BIN_DIR)$(NODE_FILE)
	rm -f $(BIN_DIR)$(TAR_FILE)
