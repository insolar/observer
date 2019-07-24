#!/usr/bin/env bash

# Changeable environment variables (parameters)
INSOLAR_ARTIFACTS_DIR=${INSOLAR_ARTIFACTS_DIR:-".artifacts"}/
LAUNCHNET_BASE_DIR=${LAUNCHNET_BASE_DIR:-"${INSOLAR_ARTIFACTS_DIR}launchnet"}/

BIN_DIR=bin
INSOLARD=$BIN_DIR/insolard
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

generate_insolard_configs()
{
    echo "generate configs"
    set -x
    go run scripts/insolard/gen/config/generate_insolar_configs.go
    { set +x; } 2>/dev/null
}

generate_keys()
{
    echo "generate configs"
    set -x
    go run scripts/insolard/gen/keys/generate_keys.go
    { set +x; } 2>/dev/null
}

check_working_dir
generate_insolard_configs
generate_keys

echo "start observer node"
${INSOLARD} \
    --config ${LAUNCHNET_BASE_DIR}insolard.yaml \
    --trace ${NODE_LOGS}/output.log
echo "observer node started in background"
