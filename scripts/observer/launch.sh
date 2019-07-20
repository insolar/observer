#!/usr/bin/env bash

# Changeable environment variables (parameters)
OBSERVER_ARTIFACTS_DIR=${OBSERVER_ARTIFACTS_DIR:-".artifacts"}/
LAUNCHNET_BASE_DIR=${LAUNCHNET_BASE_DIR:-"${OBSERVER_ARTIFACTS_DIR}launchnet"}/

BIN_DIR=bin
OBSERVER=$BIN_DIR/observer
NODE_LOGS=${LAUNCHNET_BASE_DIR}logs

mkdir -p ${NODE_LOGS}

check_working_dir()
{
    echo "check_working_dir() starts ..."
    if ! pwd | grep -q "src/github.com/insolar/observer$"
    then
        echo "Run me from insolar root"
        exit 1
    fi
    echo "check_working_dir() end."
}

generate_observer_configs()
{
    echo "generate configs"
    set -x
    go run scripts/observer/gen/config/generate_configs.go
    { set +x; } 2>/dev/null
}

generate_keys()
{
    echo "generate keys"
    set -x
    go run scripts/observer/gen/keys/generate_keys.go
    { set +x; } 2>/dev/null
}

check_working_dir
generate_observer_configs
generate_keys

echo "start observer node"
${OBSERVER} \
    --config ${LAUNCHNET_BASE_DIR}observer.yaml \
    --trace ${NODE_LOGS}/output.log
echo "observer node started in background"
