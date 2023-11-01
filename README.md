# simple-webserver
Simple webserver with few metrics for Devs

```bash
docker-compose up --build
```

1) Access localhost:8080/messages to check incoming kafka messages produced by python producer
2) Access localhost:8080/wtf and reload it multiple time to generate failures(50% chance to fail)
3) Access localhost:8080/metrics to check metrics
4) Access localhost:9090 - prometheus server UI
