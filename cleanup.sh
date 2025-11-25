#!/bin/bash

source env.sh

SERVICE_NAME="apihub-swaggerhub-plugin"
SA_NAME="apihub-swaggerhub-sa"
SA_EMAIL="${SA_NAME}@${GOOGLE_CLOUD_PROJECT}.iam.gserviceaccount.com"
SCHEDULER_JOB="apihub-swaggerhub-sync-job"
SECRET_NAME="swaggerhub-api-key"
PLUGIN_ID="swaggerhub-plugin"

echo "Deleting Cloud Scheduler job..."
gcloud scheduler jobs delete $SCHEDULER_JOB --location=$GOOGLE_CLOUD_REGION --quiet

echo "Deleting Cloud Run service..."
gcloud run services delete $SERVICE_NAME --region=$GOOGLE_CLOUD_REGION --quiet

echo "Deleting Service Account and IAM bindings..."
gcloud projects remove-iam-policy-binding $GOOGLE_CLOUD_PROJECT --member="serviceAccount:$SA_EMAIL" --role="roles/apihub.admin"
gcloud projects remove-iam-policy-binding $GOOGLE_CLOUD_PROJECT --member="serviceAccount:$SA_EMAIL" --role="roles/secretmanager.secretAccessor"
gcloud projects remove-iam-policy-binding $GOOGLE_CLOUD_PROJECT --member="serviceAccount:$SA_EMAIL" --role="roles/run.invoker"
gcloud iam service-accounts delete $SA_EMAIL --quiet

echo "Deleting Secret..."
gcloud secrets delete $SECRET_NAME --quiet

echo "Deleting Plugin Instances and Plugin..."
TOKEN=$(gcloud auth print-access-token)

# Get instances
INSTANCES=$(curl -s --location "https://apihub.googleapis.com/v1/projects/$GOOGLE_CLOUD_PROJECT/locations/$GOOGLE_CLOUD_REGION/plugins/$PLUGIN_ID/instances" \
--header "Authorization: Bearer $TOKEN" | jq -r '.pluginInstances[].name')

for INSTANCE in $INSTANCES; do
    echo "Deleting instance $INSTANCE..."
    curl -X DELETE -s --location "https://apihub.googleapis.com/v1/$INSTANCE" \
    --header "Authorization: Bearer $TOKEN"
done

echo "Deleting Plugin..."
curl -X DELETE -s --location "https://apihub.googleapis.com/v1/projects/$GOOGLE_CLOUD_PROJECT/locations/$GOOGLE_CLOUD_REGION/plugins/$PLUGIN_ID" \
--header "Authorization: Bearer $TOKEN"

echo "Cleanup complete."
