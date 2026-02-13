# Probes and Health Endpoints – What We Did, Problems, and Fixes

## 1. What We Did

### 1.1 Added Health Endpoints to All Services
- **GraphQL**: `/health` and `/ready` on port 8080 (simple process check).
- **Account Service**: `/health` and `/ready` on port 8080; `/ready` checks PostgreSQL via `repo.Ping()`.
- **Order Service**: Same pattern; `/ready` checks PostgreSQL.
- **Product Service**: Same pattern; `/ready` checks Elasticsearch via `ClusterHealth()` (not `Ping("")`).
- All gRPC services run the HTTP health server in a **goroutine** on port 8080; gRPC stays on 8081/8082/8083.

### 1.2 Layered Probe Strategy
- **DB layer (account-db, order-db, product-db)**  
  - Readiness + liveness so the DB is only in Service when it’s ready and gets restarted if it dies.  
  - PostgreSQL: `pg_isready` exec probe.  
  - Elasticsearch: HTTP `/_cluster/health?wait_for_status=yellow&timeout=1s`.
- **App layer (account, order, product, graphql)**  
  - Startup probe: allow up to ~5 min for app to start.  
  - Readiness probe: `/ready` – remove from traffic if DB/deps are down.  
  - Liveness probe: `/health` – restart if process is dead.  
  - So: **DB readiness** = “is DB accepting connections?”; **app readiness** = “can this service handle requests (e.g. DB reachable)?”.

### 1.3 Exposed Health Port on Services
- Account, Order, and Product **Services** only exposed gRPC ports initially.
- We added a second port **8080** (named `health`) so `kubectl port-forward svc/<name> 9xxx:8080` and probes can hit `/health` and `/ready`.

### 1.4 Ingress and Local Access
- Ingress forwards `/` to **graphql:8080**, so `http://localhost/health` and `http://localhost/ready` hit the GraphQL service when using ingress (usually port 80).
- Other services’ health endpoints are reached via **port-forward** or from inside the cluster (e.g. `wget http://account-service:8080/health` from a debug pod).

---

## 2. How We Did It

### 2.1 Code Changes
- **Account / Order**: Added `Ping()` to repository interface and implementation; `/ready` calls `repo.Ping()` and returns 503 on error.
- **Product**: Added `Ping(ctx)` that uses **ClusterHealth()** (not `Ping("")`); `/ready` uses it.
- **Product startup**: Health HTTP server is started **before** the Elasticsearch retry loop so port 8080 is always listening; `/ready` returns 503 until ES is connected (with `repoMu` for safe access to `repo`).
- **GraphQL**: Already HTTP; added `/health` and `/ready` handlers.

### 2.2 Kubernetes Manifests
- **Deployments**: Added `readinessProbe` and (where missing) fixed **livenessProbe** so timing fields (`initialDelaySeconds`, `periodSeconds`, `timeoutSeconds`, `failureThreshold`) are on the **probe** object, not inside `httpGet`.
- **DB deployments**: Added readiness + liveness (exec for Postgres, httpGet for Elasticsearch).
- **Services**: Added port 8080 (health) to account-service, order-service, product-service.

---

## 3. Problems We Hit and How We Overcame Them

### 3.1 Probe Spec: “unknown field” (e.g. under httpGet)
- **Symptom**: `strict decoding error: unknown field "spec.template.spec.containers[0].livenessProbe.httpGet.initialDelaySeconds"` (and similar).
- **Cause**: `initialDelaySeconds`, `periodSeconds`, `timeoutSeconds`, `failureThreshold` were placed **inside** `httpGet` instead of on the probe.
- **Fix**: Moved these fields to the same level as `httpGet` (directly under `livenessProbe` / `readinessProbe`).

### 3.2 Port-Forward “connection refused” to product-service:8080
- **Symptom**: Port-forward to product-service 9xxx:8080 worked for a moment then “connection refused” inside the pod.
- **Cause**: Health HTTP server was started **after** `retry.ForeverSleep(Elasticsearch)`. If ES was slow or down, the process never reached the line that starts the HTTP server, so nothing listened on 8080.
- **Fix**: Start the health server **first** (in a goroutine), then run the ES retry loop; `/ready` returns 503 until `repo` is set and `Ping` succeeds.

### 3.3 /ready Always 503 for product-service (ES reachable and healthy)
- **Symptom**: ES was up and K8s probes to product-db were OK, but product-service `/ready` still returned 503.
- **Cause**: `repo.Ping(ctx)` used `r.client.Ping("").Do(ctx)`. With an empty URL, the olivere/elastic client’s `Ping` was failing even when the cluster was healthy.
- **Fix**: Replaced `Ping("")` with **ClusterHealth()** and treated `green`/`yellow` as healthy (same idea as the K8s ES probe).

### 3.4 Ingress /health/account etc. Returning 404
- **Symptom**: `curl http://localhost/health/account` → 404.
- **Cause**: Ingress was forwarding path as-is; backends expect `/health` and `/ready`, not `/health/account`. Path rewrite was either wrong or not applied.
- **Fix**: We kept ingress simple: only graphql is exposed at `/` (so `localhost/health` and `localhost/ready` work for graphql). Other services’ health is reached via port-forward or from inside the cluster.

### 3.5 Port 9081 (or similar) “address already in use”
- **Symptom**: `kubectl port-forward svc/account-service 9081:8080` failed with “bind: address already in use”.
- **Cause**: Previous port-forward or another process was still using that port.
- **Fix**: Use another port (e.g. 9085) or free the port: `lsof -ti :9081 | xargs kill -9`, then retry.

### 3.6 order deployment: env vars under wrong key
- **Symptom**: `ACCOUNT_SERVICE_URL` and `PRODUCT_SERVICE_URL` looked like they were siblings of `startupProbe` instead of under `env`.
- **Cause**: YAML indentation; those two entries were not under the container’s `env` list.
- **Fix**: Moved them under the same `env` block as the DB vars.

---

## 4. Takeaways

- **Readiness** = “Should this pod receive traffic?” (e.g. DB/deps OK). Failure → pod removed from Service, **no restart**.
- **Liveness** = “Is this process alive?”. Failure → **restart**.
- **Startup** = “Give the app time to start”; once it succeeds, readiness and liveness take over.
- Probes don’t talk to each other; kubelet runs them and connects **directly to the pod** (so liveness still works when the pod is not in Service endpoints).
- DB and app both have readiness so each layer can protect itself; app readiness can depend on DB without duplicating “is DB up?” in a wrong place.
- For Elasticsearch, **ClusterHealth()** is more reliable than **Ping("")** for “is the cluster usable?” in Go.

---

## 5. File / Doc References

- **Commands (probes, health, port-forward, rollout):** `docs/PROBES_COMMANDS_REFERENCE.md`
- **Manifests:**  
  - Deployments: `k8s/account/deployment.yaml`, `k8s/order/deployment.yaml`, `k8s/product/deployment.yaml`, `k8s/graphql/deployment.yaml`, `k8s/*/deployment.yaml` for DBs.  
  - Services: `k8s/account/service.yaml`, etc.  
  - Ingress: `k8s/ingress/ingress.yaml`.
