#!/bin/bash
set -e

RELEASE=${1:-latest}
CONTAINER_NAME="lgtm"
IMAGE_NAME="docker.io/grafana/otel-lgtm:${RELEASE}"

echo "Starting Guardian monitoring environment..."

# Create necessary directories if they don't exist
mkdir -p "$PWD"/container/grafana/dashboards
mkdir -p "$PWD"/container/prometheus
mkdir -p "$PWD"/container/loki

# Copy the Guardian dashboard to the Grafana dashboards directory
cp -f /Users/rohanadwankar/guardian/guardian-dashboard.json "$PWD"/container/grafana/dashboards/ 2>/dev/null || echo "Warning: Guardian dashboard not found, using existing configuration"

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

# Check if container is already running
if docker ps -q --filter "name=${CONTAINER_NAME}" | grep -q .; then
  echo "LGTM container is already running"
  echo "To access Grafana: http://localhost:3000"
  echo "OTel GRPC endpoint: localhost:4317"
  echo "OTel HTTP endpoint: localhost:4318"
  exit 0
fi

# Check if container exists but is stopped
if docker ps -a -q --filter "name=${CONTAINER_NAME}" | grep -q .; then
  echo "Restarting existing LGTM container..."
  docker start "${CONTAINER_NAME}"
  echo "LGTM container restarted"
  echo "To access Grafana: http://localhost:3000"
  echo "OTel GRPC endpoint: localhost:4317"
  echo "OTel HTTP endpoint: localhost:4318"
  exit 0
fi

# Check if image exists, pull only if needed
if ! docker image inspect "${IMAGE_NAME}" >/dev/null 2>&1; then
  echo "Pulling Docker image ${IMAGE_NAME}..."
  docker pull "${IMAGE_NAME}"
else
  echo "Using existing Docker image ${IMAGE_NAME}"
fi

echo "Starting LGTM container..."
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
  --env-file .env \
  "${IMAGE_NAME}"

echo "LGTM container started successfully!"
echo "To access Grafana: http://localhost:3000"
echo "OTel GRPC endpoint: localhost:4317"
echo "OTel HTTP endpoint: localhost:4318"
echo "Use 'docker stop ${CONTAINER_NAME}' to stop the container"