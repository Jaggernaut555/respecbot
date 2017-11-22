#!/bin/bash
curl -H "Content-Type: application/json" -X POST -d '{"token":"$DEPLOY_TOKEN"}' http://jaggernaut.ca:9000/hooks/deploy-respecbot-webhook