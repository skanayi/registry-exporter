# registry-exporter
A custom prometheus exporter to monitor docker registry by building,pulling and pushing docker images
# Usage
- clone repo
- go build
- set environment variables
- run ./registry-exporter ,make sure you are running this from a machine with docker installed
- you can  see the metrics at http://localhost:9300/metrics
- configure prometheus to scrape metrics from metrics endpoint
