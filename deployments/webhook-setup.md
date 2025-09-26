# Webhook Setup Guide

This guide explains how to configure GitHub webhooks in each service repository to trigger deployments via the isA_Cloud deployment center.

## For Each Service Repository

Add the following workflow file to trigger deployments when code is pushed:

### 1. isA_MCP Repository
Create `.github/workflows/deploy-trigger.yml`:

```yaml
name: Trigger Deployment

on:
  push:
    branches: [main]

jobs:
  trigger-deploy:
    runs-on: ubuntu-latest
    steps:
    - name: Trigger deployment in isA_Cloud
      uses: peter-evans/repository-dispatch@v2
      with:
        token: ${{ secrets.DEPLOY_TOKEN }}
        repository: xenophobed/isA_Cloud
        event-type: deploy-trigger
        client-payload: |
          {
            "repository": "isA_MCP",
            "sha": "${{ github.sha }}",
            "ref": "${{ github.ref }}"
          }
```

### 2. isA_Model Repository
Same workflow but change `repository: "isA_MCP"` to `repository: "isA_Model"`

### 3. isA_Agent Repository
Same workflow but change `repository: "isA_MCP"` to `repository: "isA_Agent"`

### 4. isA_user Repository
Same workflow but change `repository: "isA_MCP"` to `repository: "isA_user"`

### 5. isA_Chain Repository
Same workflow but change `repository: "isA_MCP"` to `repository: "isA_Chain"`

## Required GitHub Secrets

### In isA_Cloud repository:
- `AWS_ACCESS_KEY_ID` - AWS access key
- `AWS_SECRET_ACCESS_KEY` - AWS secret key
- `SLACK_WEBHOOK_URL` - Slack webhook for notifications (optional)

### In each service repository:
- `DEPLOY_TOKEN` - GitHub Personal Access Token with repo permissions

## Setup Steps

1. Create a GitHub Personal Access Token:
   - Go to GitHub Settings > Developer settings > Personal access tokens
   - Generate new token with `repo` scope
   - Copy the token

2. Add `DEPLOY_TOKEN` secret to each service repository:
   - Go to repository Settings > Secrets and variables > Actions
   - Click "New repository secret"
   - Name: `DEPLOY_TOKEN`
   - Value: Your GitHub token

3. Add the workflow file to each service repository

4. Push changes to trigger the setup

## Manual Deployment

You can also manually trigger deployments from the isA_Cloud repository:
- Go to Actions tab
- Select "Deploy Services" workflow
- Click "Run workflow"
- Choose service and environment
- Click "Run workflow"

## Testing

To test the setup:
1. Make a change to any service repository
2. Push to main branch
3. Check isA_Cloud repository Actions tab
4. Verify deployment triggered automatically