#!/bin/bash

SERVICE_NAME="apihub-swaggerhub-plugin"

source env.sh

for var in GOOGLE_CLOUD_PROJECT GOOGLE_CLOUD_REGION SWAGGERHUB_OWNER SWAGGERHUB_API_KEY; do
  if [ -z "${!var}" ]; then
    echo "Error: $var is not set. Please set it in env.sh"
    exit 1
  fi
done

gcloud config set project $GOOGLE_CLOUD_PROJECT
gcloud services enable secretmanager.googleapis.com run.googleapis.com cloudbuild.googleapis.com cloudscheduler.googleapis.com
TOKEN=$(gcloud auth print-access-token)

# Create plugin
curl --location "https://apihub.googleapis.com/v1/projects/$GOOGLE_CLOUD_PROJECT/locations/$GOOGLE_CLOUD_REGION/plugins?pluginId=swaggerhub-plugin" \
--header 'Content-Type: application/json' \
--header "Authorization: Bearer $TOKEN" \
--data '{
    "displayName": "SwaggerHub",
    "description": "SwaggerHub onramp plugin.",
    "ownershipType": "USER_OWNED",
    "actionsConfig": [
        {
            "id": "sync-action",
            "displayName": "Sync API metadata",
            "description": "Sync API metadata",
            "triggerMode": "NON_API_HUB_MANAGED"
        }
    ],
    "pluginCategory": "API_PRODUCER"
}'

# Create plugin instance
DATA=$(cat <<EOF
{
    "displayName": "$SWAGGERHUB_OWNER",
    "actions": [
        {
            "actionId": "sync-action",
            "curationConfig": {
                "curationType": "DEFAULT_CURATION_FOR_API_METADATA"
            }
        }
    ]
}
EOF
)

RESPONSE=$(curl -s --location "https://apihub.googleapis.com/v1/projects/$GOOGLE_CLOUD_PROJECT/locations/$GOOGLE_CLOUD_REGION/plugins/swaggerhub-plugin/instances" \
--header 'Content-Type: application/json' \
--header "Authorization: Bearer $TOKEN" \
--data "$DATA")

INSTANCE_ID=$(echo $RESPONSE | jq -r '.metadata.target | split("/")[-1]')

# Create secret
if ! gcloud secrets describe swaggerhub-api-key >/dev/null 2>&1; then
  gcloud secrets create swaggerhub-api-key --replication-policy="automatic"
fi
echo -n "$SWAGGERHUB_API_KEY" | gcloud secrets versions add swaggerhub-api-key --data-file=-

SA_NAME="apihub-swaggerhub-sa"
SA_EMAIL="${SA_NAME}@${GOOGLE_CLOUD_PROJECT}.iam.gserviceaccount.com"

# Create service account
if ! gcloud iam service-accounts describe $SA_EMAIL >/dev/null 2>&1; then
  gcloud iam service-accounts create $SA_NAME --display-name="API Hub SwaggerHub Plugin Service Account"
fi

gcloud projects add-iam-policy-binding $GOOGLE_CLOUD_PROJECT \
  --member="serviceAccount:$SA_EMAIL" \
  --role="roles/apihub.admin"

gcloud projects add-iam-policy-binding $GOOGLE_CLOUD_PROJECT \
  --member="serviceAccount:$SA_EMAIL" \
  --role="roles/secretmanager.secretAccessor"

gcloud projects add-iam-policy-binding $GOOGLE_CLOUD_PROJECT \
  --member="serviceAccount:$SA_EMAIL" \
  --role="roles/run.invoker"

# Build and deploy
gcloud builds submit --tag gcr.io/$GOOGLE_CLOUD_PROJECT/$SERVICE_NAME

gcloud run deploy $SERVICE_NAME \
  --image gcr.io/$GOOGLE_CLOUD_PROJECT/$SERVICE_NAME \
  --region $GOOGLE_CLOUD_REGION \
  --port 8080 \
  --platform managed \
  --set-env-vars SWAGGERHUB_OWNER=$SWAGGERHUB_OWNER,GOOGLE_CLOUD_PROJECT=$GOOGLE_CLOUD_PROJECT,GOOGLE_CLOUD_REGION=$GOOGLE_CLOUD_REGION \
  --set-secrets SWAGGERHUB_API_KEY=swaggerhub-api-key:latest \
  --service-account $SA_EMAIL

SERVICE_URL=$(gcloud run services describe $SERVICE_NAME --region $GOOGLE_CLOUD_REGION --format 'value(status.url)')

# Create scheduler job
gcloud scheduler jobs create http apihub-swaggerhub-sync-job \
    --schedule="*/5 * * * *" \  # Run every 5 minutes
    --uri="${SERVICE_URL}/sync?plugin_instance=${INSTANCE_ID}" \
    --http-method=POST \
    --location=$GOOGLE_CLOUD_REGION \
    --oidc-service-account-email=$SA_EMAIL \
    --oidc-token-audience=$SERVICE_URL