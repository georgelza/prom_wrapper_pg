# prom_wrapper_pg

Push Gateway method...
Metrics specified as part of a struct

- Start Prometheus
docker run \
    -p 9090:9090 \
    -v /Users/george/Desktop/ProjectsCommon/prometheus/config:/etc/prometheus \
    prom/prometheus

- Start Prometheus Push Gateway
docker run -p 9091:9091 prom/pushgateway

- Start Grafana
docker run -p 3000:3000 grafana/grafana-enterprise

Included is a rough grafana dashboard, see json file
