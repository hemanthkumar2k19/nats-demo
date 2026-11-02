# NATS CLI and Monitoring Commands

Collection of commonly used NATS CLI commands and monitoring endpoints for environment setup, inspection, benchmarking, and health checks.

## Context Management

Add a named NATS CLI context pointing to the target server:

```bash
nats context add dev --server nats://<NATS_SERVER_IP>:4222
```

Switch active CLI context to `dev`:

```bash
nats context select dev
```

## Cluster and Stream Inspection

Report current NATS server statistics, resource usage, and JetStream status across the cluster:

```bash
nats server report
```

List all available JetStream streams:

```bash
nats stream ls
```

## Workload Benchmarking

Run a Core NATS publisher benchmark on the subject `benchmark.core`:

```bash
nats bench pub benchmark.core
```

## Monitoring Endpoints

Query the NATS server monitoring endpoint for general server metrics, memory, CPU, and connection counts in JSON format:

```bash
curl http://<NATS_SERVER_IP>:8222/varz
```

```