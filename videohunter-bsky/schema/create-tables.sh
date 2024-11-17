#!/bin/sh
echo "Creating tables..."
aws dynamodb create-table --cli-input-json file://video.json --endpoint-url http://dynamodb:8000 --region us-east-1
aws dynamodb create-table --cli-input-json file://settings.json --endpoint-url http://dynamodb:8000 --region us-east-1