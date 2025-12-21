---
description: How to deploy the backend to GCP Cloud Run
---

# Deploying Backend to GCP Cloud Run

This guide explains how to deploy the containerized Go backend to Google Cloud Run.

## Prerequisites
1.  [Google Cloud SDK (gcloud)](https://cloud.google.com/sdk/docs/install) installed and initialized.
2.  A Google Cloud Project with billing enabled.
3.  The following APIs enabled:
    - Cloud Run API
    - Artifact Registry API
    - Cloud Build API

## Step-by-Step Deployment

### 1. Create an Artifact Registry Repository
Create a repository to store your Docker images:
```bash
gcloud artifacts repositories create family-potluck-repo \
    --repository-format=docker \
    --location=us-central1 \
    --description="Docker repository for Family Potluck"
```

### 2. Build and Push the Image using Cloud Build
Run this command from the `backend` directory. It will build the image in the cloud and push it to your repository:
```bash
gcloud builds submit --tag us-central1-docker.pkg.dev/[PROJECT_ID]/family-potluck-repo/backend:latest .
```
*Replace `[PROJECT_ID]` with your actual GCP Project ID.*

### 3. Deploy to Cloud Run
Deploy the image to Cloud Run, setting the necessary environment variables:
```bash
gcloud run deploy family-potluck-backend \
    --image us-central1-docker.pkg.dev/[PROJECT_ID]/family-potluck-repo/backend:latest \
    --platform managed \
    --region us-central1 \
    --allow-unauthenticated \
    --set-env-vars="MONGODB_URI=your_mongodb_uri,MONGODB_DATABASE=familypotluck,JWT_SECRET=your_secret,GOOGLE_CLIENT_ID=your_client_id,ALLOWED_ORIGINS=https://your-frontend.web.app"
```

## Managing Secrets
For production, it is highly recommended to use **Secret Manager** instead of plain environment variables:
1.  Create secrets in Secret Manager.
2.  Grant the Cloud Run service account access to these secrets.
3.  Reference them in the deploy command using `--set-secrets`.

## Updating the Deployment
To deploy a new version, simply repeat steps 2 and 3. Cloud Run will automatically handle the traffic transition to the new revision.
