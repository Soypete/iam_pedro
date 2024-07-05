#!/bin/bash
#
#
# This runs prometheus in a container of our pi4
podman run -p "9090:9090" -v ./prometheus.yml:/etc/prometheus/prometheus.yml -v prometheus-data:/prometheus prom/prometheus
