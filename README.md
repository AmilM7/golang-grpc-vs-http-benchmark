# Golang gRPC vs REST Benchmark

This repository represents my personal project, built out of curiosity for gRPC. It provided a simple user service implemented over both HTTP/JSON and gRPC transports and a additional benchmark client. The goal is to compare request/response performance between gRPC and REST across multiple deployment scenarios:

- Local machine (server and client on the same host)
- Virtual machines on the same physical host
- Two separate physical machines

Both transports use the same sequence of create, update, get, delete  operations against an in-memory user store. The benchmark client executes create -> updat -> ge ->delete sequences for both HTTP and gRPC to capture latency statistics per operation.

## Project Layout

- `cmd/server`: Combined HTTP and gRPC server binary. 
- `cmd/testclient`: Benchmark driver binary.
- `internal/*`: Domain logic, transports, and configuration helpers.
- `proto`: Protobuf definitions; generated Go stubs under `pkg/gen`.

## Building

If this is your first time cloning the repository, you must generate the gRPC and Protobuf Go bindings before building:

```sh
mkdir -p pkg/gen/user/v1
make proto
```

Build: 

```sh
make build       # build the server binary at bin/server
make build-test  # build the benchmark client at bin/testclient
```

## Running the Server

```sh
make run
```

There are the following environment variables which control the listener addresses, which are useful for different variances of testing:

- `HTTP_HOST` (default `127.0.0.1`)
- `HTTP_PORT` (default `8087`)
- `GRPC_HOST` (default `127.0.0.1`)
- `GRPC_PORT` (default `50055`)
- `SHUTDOWN_GRACE_SECONDS` (optional, default `5`)

Example (server bound to a public interface):

```sh
make run HTTP_HOST=0.0.0.0 GRPC_HOST=0.0.0.0
```

When deploying the binary manually, export the same variables before running `bin/server`.

## Running the Benchmark Client

```sh
make run-test
```

There are the following environment variables:

- `BENCH_HTTP_BASE_URL` (default built from `HTTP_HOST`/`HTTP_PORT`, e.g. `http://127.0.0.1:8087`)
- `BENCH_GRPC_ADDR` (default built from `GRPC_HOST`/`GRPC_PORT`, e.g. `127.0.0.1:50055`)
- `BENCH_ITERATIONS` (default `100`, applied per operation)
- `BENCH_CONCURRENCY` (number of parallel workers per transport, default `5`)
- `BENCH_WARMUP` (how many warm-up create/delete cycles to issue before measuring, default `20`)
- `BENCH_RPC_TIMEOUT_MS` (per-request deadline in milliseconds, default `2000`)

To target a remote server:

```sh
make run-test BENCH_HTTP_BASE_URL=http://10.0.0.5:8087 BENCH_GRPC_ADDR=10.0.0.5:50055
```

The benchmark output lists per-operation latency statistics (average, minimum, and maximum) for both transports.

### Payload realism

Each request carries a richer, more realistic user profile designed to stress serialization overhead:

- Contact data (`phone`, `address`) and a long-form `bio`
- Repeated tags to add moderately sized collections
- A 4 KiB synthetic `avatar` byte slice to exercise binary payload handling

The same schema is used by both HTTP/JSON and gRPC/protobuf so comparisons remain apples-to-apples while involving more realistic payload sizes.

#### 📊 Results

The following results summarize **average (avg)**, **minimum (min)**, and **maximum (max)** latency measurements across three environments.

---

### 🖥️ Local Machine (Same Host)

| Operation | **HTTP avg** | **HTTP min** | **HTTP max** | **gRPC avg** | **gRPC min** | **gRPC max** |
|:-----------|-------------:|-------------:|-------------:|-------------:|-------------:|-------------:|
| Create | 730.007 µs | 334.285 µs | 1.809 ms | **264.14 µs** | 148.37 µs | 608.64 µs |
| Update | 681.298 µs | 324.268 µs | 1.392 ms | **280.29 µs** | 151.28 µs | 681.28 µs |
| Get | 430.761 µs | 208.102 µs | 1.054 ms | **279.11 µs** | 140.23 µs | 701.01 µs |
| Delete | 238.745 µs | 84.674 µs | 636.29 µs | **242.03 µs** | 113.56 µs | 567.93 µs |

**Observation:**  
Since there’s no real network latency (loopback), serialization efficiency dominates.  
gRPC is ~2–3× faster due to **binary Protobuf** and **HTTP/2 multiplexing**, whereas JSON/HTTP suffers from text encoding overhead.

---

### 💻 Virtual Machines (Same Physical Host)

| Operation | **HTTP avg** | **HTTP min** | **HTTP max** | **gRPC avg** | **gRPC min** | **gRPC max** |
|:-----------|-------------:|-------------:|-------------:|-------------:|-------------:|-------------:|
| Create | 31.49 ms | 0.66 ms | 96.92 ms | **8.33 ms** | 1.88 ms | 21.70 ms |
| Update | 19.88 ms | 0.78 ms | 88.57 ms | **6.88 ms** | 1.01 ms | 17.69 ms |
| Get | 14.08 ms | 0.41 ms | 70.68 ms | **4.18 ms** | 0.38 ms | 11.28 ms |
| Delete | 16.32 ms | 0.17 ms | 89.92 ms | **3.78 ms** | 0.68 ms | 12.39 ms |

**Observation:**  
Even with virtualized networking (virtio/vnet), latency remains < 1 ms.  
gRPC performs **3–4× faster** thanks to its **persistent HTTP/2 streams** versus multiple TCP handshakes in HTTP/1.1.  
This scenario closely represents **microservices within the same host**, where gRPC shines.

---

### 🌐 Cross-Host (Two Physical Machines)

#### Config #1 → `iterations=10000`, `concurrency=20`, `warmup=10`

| Operation | **HTTP avg** | **HTTP min** | **HTTP max** | **gRPC avg** | **gRPC min** | **gRPC max** |
|:-----------|-------------:|-------------:|-------------:|-------------:|-------------:|-------------:|
| Create | 59.30 ms | 30.89 ms | 922.82 ms | **50.25 ms** | 30.38 ms | 506.29 ms |
| Update | 58.68 ms | 30.84 ms | 844.20 ms | **54.30 ms** | 29.74 ms | 545.08 ms |
| Get | 46.50 ms | 28.41 ms | 546.88 ms | **46.65 ms** | 27.52 ms | 521.41 ms |
| Delete | 43.09 ms | 26.33 ms | 560.94 ms | **39.92 ms** | 26.65 ms | 475.42 ms |

#### Config #2 → `iterations=1000`, `concurrency=50`, `warmup=30`

| Operation | **HTTP avg** | **HTTP min** | **HTTP max** | **gRPC avg** | **gRPC min** | **gRPC max** |
|:-----------|-------------:|-------------:|-------------:|-------------:|-------------:|-------------:|
| Create | 157.25 ms | 36.27 ms | 832.58 ms | **115.34 ms** | 39.14 ms | 547.50 ms |
| Update | 148.84 ms | 34.94 ms | 285.55 ms | **142.96 ms** | 48.42 ms | 507.20 ms |
| Get | 106.13 ms | 29.60 ms | 256.03 ms | **100.39 ms** | 31.53 ms | 269.56 ms |
| Delete | 84.22 ms | 29.88 ms | 455.32 ms | **71.11 ms** | 28.17 ms | 185.53 ms |

**Observation:**  
Across a real network (~30–40 ms latency), total RTT dominates.  
Serialization overhead becomes negligible; both protocols spend most time waiting on network I/O.  
gRPC remains **slightly faster** due to persistent multiplexed connections and binary encoding, but the performance gap shrinks because latency and TCP congestion control overshadow serialization gains.  
At higher concurrency (`BENCH_CONCURRENCY=50`), performance differences fluctuate based on queueing and network conditions.

---

## 🧩 Conclusion

Across all environments:

- gRPC consistently outperforms HTTP/JSON in **low-latency** or **virtualized** setups where CPU and serialization costs dominate.  
- In real distributed scenarios, where **network latency** is the bottleneck, both converge in performance.  
- gRPC’s key advantage lies not only in raw speed but in its **connection persistence**, **multiplexing**, and **binary efficiency**, making it ideal for **microservice communication** within datacenters or clusters.


