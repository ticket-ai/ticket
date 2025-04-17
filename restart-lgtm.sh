#!/bin/bash
set -e

RELEASE=${1:-latest}
CONTAINER_NAME="lgtm"
IMAGE_NAME="docker.io/grafana/otel-lgtm:${RELEASE}"
# Define the absolute path to the dashboard file
DASHBOARD_SOURCE_PATH="/Users/rohanadwankar/guardian/guardian-dashboard.json"
DASHBOARD_TARGET_PATH="/data/grafana/dashboards/guardian-dashboard.json"

echo "Starting Guardian monitoring environment..."

# Check if the container exists and stop/remove it if it does
if docker ps -a --filter "name=${CONTAINER_NAME}" --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
  echo "Found existing container '${CONTAINER_NAME}'. Stopping and removing it..."
  docker stop "${CONTAINER_NAME}" > /dev/null
  docker rm "${CONTAINER_NAME}" > /dev/null
  echo "Existing container '${CONTAINER_NAME}' stopped and removed."
else
  echo "No existing container named '${CONTAINER_NAME}' found."
fi

# Create necessary directories if they don't exist
# Note: We still need the target directory structure for the volume mount
mkdir -p "$PWD"/container/grafana/dashboards
mkdir -p "$PWD"/container/prometheus
mkdir -p "$PWD"/container/loki

# Create provisioning directory for Grafana if it doesn't exist
mkdir -p "$PWD"/container/grafana/provisioning/dashboards

# Create a dashboard provisioning configuration file if it doesn't exist
if [ ! -f "$PWD"/container/grafana/provisioning/dashboards/guardian-dashboards.yaml ]; then
  cat > "$PWD"/container/grafana/provisioning/dashboards/guardian-dashboards.yaml << EOF
apiVersion: 1

providers:
  - name: 'Guardian Dashboards'
    orgId: 1
    folder: 'Guardian'
    folderUid: ''
    type: file
    disableDeletion: false
    updateIntervalSeconds: 10
    allowUiUpdates: true
    options:
      path: /data/grafana/dashboards
EOF
  echo "Created Grafana dashboard provisioning configuration"
fi

# Create .env file if it doesn't exist
if [ ! -f .env ]; then
  touch .env
  echo "Created .env file"
fi

# Check if image exists, pull only if needed
if ! docker image inspect "${IMAGE_NAME}" >/dev/null 2>&1; then
  echo "Pulling Docker image ${IMAGE_NAME}..."
  docker pull "${IMAGE_NAME}"
else
  echo "Using existing Docker image ${IMAGE_NAME}"
fi

echo "Starting LGTM container..."
# Check if the source dashboard file exists before trying to mount it
if [ ! -f "${DASHBOARD_SOURCE_PATH}" ]; then
  echo "Warning: Source dashboard file not found at ${DASHBOARD_SOURCE_PATH}. Dashboard will not be mounted."
  docker run \
    --name ${CONTAINER_NAME} \
    -p 3000:3000 \
    -p 4317:4317 \
    -p 4318:4318 \
    -d \
    -v "$PWD"/container/grafana:/data/grafana \
    -v "$PWD"/container/prometheus:/data/prometheus \
    -v "$PWD"/container/loki:/data/loki \
    -e GF_PATHS_DATA=/data/grafana \
    -e GF_PATHS_PROVISIONING=/data/grafana/provisioning \
    --env-file .env \
    "${IMAGE_NAME}"
else
  echo "Mounting dashboard from ${DASHBOARD_SOURCE_PATH}"
  docker run \
    --name ${CONTAINER_NAME} \
    -p 3000:3000 \
    -p 4317:4317 \
    -p 4318:4318 \
    -d \
    -v "$PWD"/container/grafana:/data/grafana \
    -v "$PWD"/container/prometheus:/data/prometheus \
    -v "$PWD"/container/loki:/data/loki \
    -v "${DASHBOARD_SOURCE_PATH}:${DASHBOARD_TARGET_PATH}:ro" \
    -e GF_PATHS_DATA=/data/grafana \
    -e GF_PATHS_PROVISIONING=/data/grafana/provisioning \
    --env-file .env \
    "${IMAGE_NAME}"
fi

echo "LGTM container started successfully!"
echo "To access Grafana: http://localhost:3000"
echo "OTel GRPC endpoint: localhost:4317"
echo "OTel HTTP endpoint: localhost:4318"
echo "Use 'docker stop ${CONTAINER_NAME}' to stop the container"