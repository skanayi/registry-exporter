# registry-exporter
A custom proemtheus exporter to monitor docker registry by pulling and pushing docker images
# Usage
- clone repo
- go build
- set environment variables
- run ./registry-exporter
- you can  see the metrics at http://localhost:9300/metrics
- configure prometheus to scrape metrics from metrics endpoint
