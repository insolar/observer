 #!/usr/bin/env bash


# strict mode
set -euo pipefail
IFS=$'\n\t'

# configurable vars
INSOLAR_SCRIPTS_DIR=${INSOLAR_ARTIFACTS_DIR:-"scripts"}/
LAUNCHNET_MONITOR_DIR=${LAUNCHNET_BASE_DIR:-"${INSOLAR_SCRIPTS_DIR}monitor"}/

# Used by docker-compose config. DO NOT REMOVE.
PROMETHEUS_IN_CONFIG=${LAUNCHNET_MONITOR_DIR}prometheus/prometheus.yaml

set +x

export PROMETHEUS_CONFIG_DIR=../../${LAUNCHNET_MONITOR_DIR}prometheus/

if [[ $# -lt 1 ]]; then
# 1) if started without params  pretend to be clever and do what is expected:
# * start all monitoring services


    echo "start monitoring stack"
    cd scripts/monitor/

    docker-compose down
    docker-compose up -d
    docker-compose ps

    echo "# Grafana: http://localhost:3000 admin:pass"
    echo "# Prometheus: http://localhost:9090/targets"
    echo ""
    exit
fi

# 2) with arguments just work as thin docker-compose wrapper


docker-compose $@
