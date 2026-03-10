#!/bin/bash

# Update CONSUMER_PUSH_URL on all deployed worker VMs
# Usage: ./update-consumer-url.sh "https://your-ngrok-or-cloud-run-url/pubsub/push"

set -e

if [ -z "$1" ]; then
  echo "Usage: $0 <CONSUMER_PUSH_URL>"
  echo "Example: $0 'https://abc123.ngrok-free.dev/pubsub/push'"
  exit 1
fi

CONSUMER_PUSH_URL="$1"

GCLOUD_BIN="${GCLOUD_BIN:-gcloud}"
case "${OSTYPE:-}" in
  msys*|cygwin*|win32*)
    if command -v gcloud.cmd >/dev/null 2>&1; then
      GCLOUD_BIN="$(command -v gcloud.cmd)"
    fi
    ;;
esac

# Worker configuration: array of "vm-name:zone" pairs
WORKERS=(
  "jennah-dp:asia-northeast1-a"
  "jennah-dp-2:asia-northeast1-a"
  "jennah-dp-3:asia-northeast1-b"
)

# Common configuration (must match deploy-workers.sh)
PROJECT_ID="labs-169405"
BATCH_REGION="asia-northeast1"
DB_INSTANCE="alphaus-dev"
DB_DATABASE="main"
WORKER_PORT="8081"
JOB_CONFIG_PATH="/config/job-config.json"
WORKER_LEASE_TTL_SECONDS="10"
WORKER_CLAIM_INTERVAL_SECONDS="3"
CLOUD_TASKS_QUEUE_ID="jennah-simple"
CLOUD_RUN_IMAGE_REGISTRY="gcr.io/$PROJECT_ID"
SERVICE_ACCOUNT="gcp-sa-dev-interns@$PROJECT_ID.iam.gserviceaccount.com"
IMAGE_TAG="${IMAGE_TAG:-latest}"
IMAGE="asia-docker.pkg.dev/$PROJECT_ID/asia.gcr.io/jennah-worker:$IMAGE_TAG"

# Docker run command with updated CONSUMER_PUSH_URL
DOCKER_COMMAND='docker run -d \
  --name jennah-worker \
  --restart always \
  -p 8081:8081 \
  -e BATCH_PROVIDER=gcp \
  -e BATCH_PROJECT_ID='"$PROJECT_ID"' \
  -e BATCH_REGION='"$BATCH_REGION"' \
  -e DB_PROVIDER=spanner \
  -e DB_PROJECT_ID='"$PROJECT_ID"' \
  -e DB_INSTANCE='"$DB_INSTANCE"' \
  -e DB_DATABASE='"$DB_DATABASE"' \
  -e WORKER_PORT='"$WORKER_PORT"' \
  -e JOB_CONFIG_PATH='"$JOB_CONFIG_PATH"' \
  -e WORKER_ID=WORKER_ID_PLACEHOLDER \
  -e WORKER_LEASE_TTL_SECONDS='"$WORKER_LEASE_TTL_SECONDS"' \
  -e WORKER_CLAIM_INTERVAL_SECONDS='"$WORKER_CLAIM_INTERVAL_SECONDS"' \
  -e CLOUD_TASKS_TARGET_URL=http://localhost:'"$WORKER_PORT"'/jennah.v1.DeploymentService/SubmitJob \
  -e CLOUD_TASKS_QUEUE_ID='"$CLOUD_TASKS_QUEUE_ID"' \
  -e CLOUD_RUN_ENABLED=true \
  -e CLOUD_RUN_IMAGE_REGISTRY='"$CLOUD_RUN_IMAGE_REGISTRY"' \
  -e CLOUD_TASKS_SERVICE_ACCOUNT='"$SERVICE_ACCOUNT"' \
  -e PUBSUB_ENABLED=true \
  -e PUBSUB_TOPIC_ID=jennah-job-events \
  -e PUBSUB_PROJECT_ID='"$PROJECT_ID"' \
  -e CONSUMER_PUSH_URL='"$CONSUMER_PUSH_URL"' \
  '"$IMAGE"' \
  serve'

# Function to update a single worker
update_worker() {
  local worker_config="$1"
  local vm_name="${worker_config%:*}"
  local zone="${worker_config#*:}"
  
  echo "========================================="
  echo "Updating CONSUMER_PUSH_URL on $vm_name"
  echo "========================================="
  
  # Keep the remote command on one line for cross-platform compatibility
  REMOTE_COMMANDS="set -e; \
echo 'Stopping jennah-worker container...'; \
docker stop jennah-worker 2>/dev/null || echo 'Container not running'; \
echo 'Removing jennah-worker container...'; \
docker rm jennah-worker 2>/dev/null || echo 'Container not found'; \
echo 'Starting jennah-worker with new CONSUMER_PUSH_URL...'; \
${DOCKER_COMMAND/WORKER_ID_PLACEHOLDER/$vm_name}; \
docker ps --filter name=jennah-worker --format '{{.Names}} {{.Status}}'; \
echo 'Update complete for $vm_name'"
  
  # Execute commands on remote VM
  "$GCLOUD_BIN" compute ssh "$vm_name" \
    --zone="$zone" \
    --command="$REMOTE_COMMANDS"
  
  echo "[OK] Successfully updated $vm_name"
  echo ""
}

# Main update loop
echo "Starting CONSUMER_PUSH_URL update on all workers..."
echo "New URL: $CONSUMER_PUSH_URL"
echo ""

for worker_config in "${WORKERS[@]}"; do
  update_worker "$worker_config"
done

echo "========================================="
echo "[OK] All workers updated successfully!"
echo "========================================="
