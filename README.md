Would you like me to also sketch out a Makefile + docker-compose.yml so you can spin up the pool + Prometheus + Grafana stack in one shot for local observability testing?


## 🔑 How to use

- make build → builds the example binary
- make run → runs the example directly
- make test → runs all tests under ./...
- make tidy → cleans up go.mod and go.sum
- make clean → removes the built binary
- make run-metrics → runs the example and exposes Prometheus metrics at http://localhost:2112/metrics