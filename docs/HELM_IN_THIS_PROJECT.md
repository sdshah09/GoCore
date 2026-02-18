# How We're Using Helm Charts in This Project

A narrative of how Helm charts were introduced and integrated into the GoCore project, based on our implementation journey.

---

## The Problem We Started With

**Before Helm:** The project had raw Kubernetes manifests in `k8s/`:
- `k8s/account/deployment.yaml`, `k8s/account/service.yaml`
- `k8s/order/deployment.yaml`, `k8s/order/service.yaml`
- `k8s/product/deployment.yaml`, `k8s/product/service.yaml`
- `k8s/graphql/deployment.yaml`, `k8s/graphql/service.yaml`
- `k8s/account-db/deployment.yaml`, `k8s/account-db/pvc.yaml`, `k8s/account-db/service.yaml`, `k8s/account-db/configmap.yaml`
- `k8s/order-db/...` (same pattern)
- `k8s/product-db/...` (same pattern)
- `k8s/secrets.yaml`
- `k8s/ingress/ingress.yaml`

**Issues:**
1. **Repetition:** Same patterns across services (probes, resources, env structure).
2. **Manual updates:** Changing replicas or image tags meant editing many files.
3. **No parameterization:** Hard-coded values everywhere (e.g. image tags, resource limits).
4. **Deployment workflow:** Had to `kubectl apply -f k8s/account`, `kubectl apply -f k8s/order`, etc. in order.
5. **Secrets in Git:** `k8s/secrets.yaml` had base64-encoded values that could be committed.

---

## The Solution: Helm Charts

We created a **Helm chart** (`gocore/`) that:
- **Templates** the repetitive YAML (one template generates multiple Deployments/Services).
- **Parameterizes** everything via `values.yaml` (replicas, images, ports, env, probes, resources).
- **One command** deploys everything: `helm install gocore ./gocore -f gocore/values.yaml -f gocore/values-secret.yaml`.
- **Keeps secrets out of Git** via `values-secret.yaml` (gitignored).

---

## How We Structured It

### Step 1: Created the Chart Skeleton

```bash
helm create gocore
```

This gave us:
- `Chart.yaml` (metadata)
- `values.yaml` (default scaffold)
- `templates/` (empty except `_helpers.tpl`)

### Step 2: Analyzed Existing Manifests

We read all the `k8s/*.yaml` files to extract:
- **Common patterns:** All microservices had similar structure (ports, probes, resources).
- **DB patterns:** Postgres (account-db, order-db) vs Elasticsearch (product-db) had different probes and volumes.
- **Service patterns:** Microservices had two ports (grpc + health); graphql had one; DBs had one.

### Step 3: Built values.yaml from k8s/ Manifests

We created a **comprehensive `values.yaml`** that mirrors all config from `k8s/`:

- **`services:`** partition for microservices (account, order, product, graphql)
  - Each has: `enabled`, `name`, `replicas`, `image.repository`/`tag`, `ports.grpc`/`health`, `env`, `probes`, `resources`
- **`dbServices:`** partition (list) for DB Services (used by `dbService.yaml`)
- **`accountDb`, `orderDb`, `productDb`:** Full DB config (image, port, env, probes, PVC, lifecycle, resources)
- **`configMaps:`** account-db-init, order-db-init (init SQL)
- **`ingress:`** name, className, rules
- **`secrets:`** structure only (no real values)

**Key decision:** We put microservices under `services.*` (a map) so `service.yaml` and `deployment.yaml` can range over them. DBs are separate top-level keys (`accountDb`, `orderDb`, `productDb`) because they have different structures (Postgres vs Elasticsearch).

### Step 4: Created Templates (One Resource Type Per File)

We split templates by **resource type** (not by component):

| Template | What It Renders |
|----------|----------------|
| `deployment.yaml` | All 4 microservice Deployments (account, order, product, graphql) |
| `db-deployment.yaml` | 3 PVCs + 3 DB Deployments (account-db, order-db, product-db) |
| `service.yaml` | 4 microservice Services |
| `dbService.yaml` | 3 DB Services |
| `secret.yaml` | 1 Secret (db-credentials) |
| `ingress.yaml` | 1 Ingress (gocore-ingress) |

**Why this structure?**
- **Separation of concerns:** Each template focuses on one resource type.
- **Easy to find:** Want to change how Services are rendered? Look at `service.yaml`.
- **Reusable:** The same template logic applies to all services (range over `services`).

### Step 5: Partitioned Values for Service Templates

We created **two partitions** in `values.yaml`:

1. **`microserviceServices`** (later renamed to `services`): List of service definitions for `service.yaml`
   - Each entry: `ref` (points to `services.account`), `name`, `selector`, `ports`
   - Template checks `services[.ref].enabled` to decide if Service should be created

2. **`dbServices`:** List for `dbService.yaml`
   - Each entry: `ref` (points to `accountDb`/`orderDb`/`productDb`), `name`, `selector`, `ports`
   - Template checks `accountDb.enabled` / `orderDb.enabled` / `productDb.enabled`

**Why partitions?**
- `service.yaml` needs only Service-specific data (name, selector, ports).
- The full component config (replicas, image, env, probes) lives under `services.*` or `accountDb`/`orderDb`/`productDb`.
- Partitions avoid duplication: ports are defined once in `services.account.ports`, and `service.yaml` reads from there.

**Later simplification:** We removed the `microserviceServices` partition and had `service.yaml` range directly over `services.*` (the map), deriving selector from the key (`account`, `order`, etc.) and ports from `services.account.ports.grpc` + `services.account.ports.health`. This is cleaner.

### Step 6: Secrets Management (Option B)

**Problem:** We can't put real DB credentials in `values.yaml` (it's committed to Git).

**Solution:** Two files:
- **`values-secret.yaml`** (gitignored): Real credentials (`postgres-user`, `postgres-password`)
- **`values-secret.yaml.example`** (committed): Template showing the format

**Usage:** Always pass both files:
```bash
helm install gocore ./gocore -f gocore/values.yaml -f gocore/values-secret.yaml
```

**Why `-f` twice?** Helm merges multiple `-f` files (later files override earlier ones). So `values-secret.yaml` can override just the `secrets.dbCredentials.stringData` section without duplicating all of `values.yaml`.

**`.gitignore` addition:** We added patterns like `*-secret*.yaml` and `values-*.local.yaml` so secret files are never committed.

---

## How Templates Work

### deployment.yaml

```yaml
{{- range $selector, $svc := .Values.services }}
{{- if $svc.enabled }}
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ $svc.name }}
spec:
  replicas: {{ $svc.replicas }}
  # ... container image, ports, probes, resources, env ...
{{- end }}
{{- end }}
```

**What it does:** Loops over `services` (account, order, product, graphql). For each enabled service, emits a Deployment with values from `$svc.*`.

**Env handling:** Keys ending in `_SECRET` (e.g. `DB_USER_SECRET`) become `valueFrom.secretKeyRef`; others become plain `value:`.

### db-deployment.yaml

```yaml
{{- $dbRefs := list "accountDb" "orderDb" "productDb" }}
{{- range $dbRefs }}
{{- $db := index $.Values . }}
{{- if $db.enabled }}
---
# PVC first
apiVersion: v1
kind: PersistentVolumeClaim
# ... from $db.pvc ...
---
# Then Deployment
apiVersion: apps/v1
kind: Deployment
# ... Postgres vs Elasticsearch branching ...
{{- end }}
{{- end }}
```

**What it does:** Loops over the three DB keys. For each enabled DB:
1. Emits a PVC (from `$db.pvc`).
2. Emits a Deployment: if `$db.initConfigMap` exists → Postgres (exec probes, preStop, init volume); else → Elasticsearch (httpGet probes, single data volume).

**Why PVC before Deployment?** Kubernetes needs the PVC to exist before a Pod can mount it. Helm renders in order, so PVC comes first in the output.

### service.yaml

```yaml
{{- range $selector, $svc := .Values.services }}
{{- if $svc.enabled }}
---
apiVersion: v1
kind: Service
metadata:
  name: {{ $svc.name }}
spec:
  selector:
    app: {{ $selector }}  # "account", "order", etc.
  ports:
{{- if $svc.service }}
  - port: {{ $svc.service.port }}
    targetPort: {{ $svc.service.targetPort }}
{{- else }}
  - name: grpc
    port: {{ $svc.ports.grpc }}
    targetPort: {{ $svc.ports.grpc }}
  - name: health
    port: {{ $svc.ports.health }}
    targetPort: {{ $svc.ports.health }}
{{- end }}
{{- end }}
{{- end }}
```

**What it does:** Loops over `services`. For each enabled service:
- Selector is the key (`account`, `order`, etc.).
- Ports: if `service` block exists (graphql) → single port; else → grpc + health from `ports`.

**Why declare ports in template?** We could have had a `servicePorts` list in values, but deriving from `ports.grpc`/`ports.health` avoids duplication (ports are already in `services.*` for the Deployment).

---

## Deployment Workflow

### First-Time Install

```bash
# 1. Create secret values file (if not exists)
cp gocore/values-secret.yaml.example gocore/values-secret.yaml
# Edit gocore/values-secret.yaml with your password

# 2. Install
helm install gocore ./gocore -f gocore/values.yaml -f gocore/values-secret.yaml
```

**What happens:**
- Helm reads `values.yaml` (base config).
- Helm reads `values-secret.yaml` (merges secrets).
- Helm renders all templates.
- Helm applies all manifests to Kubernetes.
- Result: 4 Deployments, 7 Services, 3 PVCs, 1 Secret, 1 Ingress.

### Updating After Changes

```bash
# Edit gocore/values.yaml (e.g. change replicas, image tag)
# Then:
helm upgrade gocore ./gocore -f gocore/values.yaml -f gocore/values-secret.yaml
```

**What happens:**
- Helm compares current release state with new templates/values.
- Helm updates changed resources (e.g. Deployment spec changes trigger a rolling update).
- PVCs are not recreated (they're persistent).

### Idempotent Install/Upgrade

```bash
helm upgrade --install gocore ./gocore -f gocore/values.yaml -f gocore/values-secret.yaml
```

**Why:** Same command works for first install and every update. Useful in scripts/CI.

---

## Challenges We Faced and How We Solved Them

### Challenge 1: "Resource Already Exists" Errors

**Problem:** Resources created with `kubectl apply -f k8s/` don't have Helm ownership labels/annotations. Helm can't adopt them.

**Error:**
```
Error: Secret "db-credentials" exists and cannot be imported: missing key "app.kubernetes.io/managed-by": must be set to "Helm"
```

**Solution:** Delete the conflicting resources, then let Helm create them:
```bash
helm uninstall gocore --ignore-not-found
kubectl delete deployment account-db order-db product-db account-service order-service product-service graphql -n default --ignore-not-found
kubectl delete ingress gocore-ingress -n default --ignore-not-found
kubectl delete pvc account-db-pvc order-db-pvc product-db-pvc -n default --ignore-not-found
kubectl delete secret db-credentials -n default --ignore-not-found
kubectl delete svc account-db order-db product-db account-service order-service product-service graphql -n default --ignore-not-found

helm install gocore ./gocore -f gocore/values.yaml -f gocore/values-secret.yaml
```

**Lesson:** Don't mix `kubectl apply` and Helm for the same resources. Choose one approach per namespace.

### Challenge 2: Apply Conflicts

**Problem:** Even after adding Helm labels/annotations, Helm's server-side apply conflicts with kubectl's client-side apply on fields like `imagePullPolicy`, `resources.limits.memory`, `spec.rules`.

**Error:**
```
conflict with "kubectl-client-side-apply" using apps/v1: .spec.template.spec.containers[name="postgres"].imagePullPolicy
```

**Solution:** Same as Challenge 1 – delete and let Helm recreate. Don't try to "adopt" resources that were created by kubectl.

### Challenge 3: Typo in Filename

**Problem:** Typo `values-secret.yamll` (extra `l`) caused Helm to not find the file or use wrong values.

**Solution:** Always use `values-secret.yaml` (one `l`). Double-check filenames in commands.

### Challenge 4: Secrets in Git

**Problem:** Initially we had default credentials in `values.yaml` (e.g. `postgres-password: "password"`). This would be committed to Git.

**Solution:** Moved secrets to `values-secret.yaml` (gitignored). `values.yaml` only has comments explaining how to provide secrets (`--set` or `-f values-secret.yaml`).

---

## Current State

**What we have:**
- ✅ Helm chart at `gocore/` with all templates
- ✅ `values.yaml` with all non-secret config (committed)
- ✅ `values-secret.yaml` for local dev credentials (gitignored)
- ✅ `values-secret.yaml.example` as template (committed)
- ✅ `.gitignore` patterns to prevent committing secrets
- ✅ One command deploys everything: `helm install` or `helm upgrade --install`

**What we can do:**
- Change replicas by editing `values.yaml` → `helm upgrade`
- Change image tags by editing `values.yaml` → `helm upgrade`
- Add new services by adding to `services.*` in `values.yaml` (templates auto-generate Deployment + Service)
- See what would be deployed: `helm template` (dry run)
- Rollback: `helm rollback gocore <revision>`

**What we avoid:**
- ❌ Editing many YAML files for one change
- ❌ Applying manifests in order manually
- ❌ Committing secrets to Git
- ❌ Mixing `kubectl apply` and Helm (causes conflicts)

---

## Benefits We Gained

1. **DRY (Don't Repeat Yourself):** One template generates multiple Deployments/Services.
2. **Parameterization:** Change one value in `values.yaml`, affects all resources.
3. **One command deployment:** `helm install` vs many `kubectl apply` commands.
4. **Secrets safety:** Secrets stay local, never committed.
5. **Versioning:** Chart version tracks changes; can rollback.
6. **Easier updates:** Change values → `helm upgrade` → done.
7. **Template validation:** `helm template` shows what would be deployed before applying.

---

## Next Steps (Optional Enhancements)

- **ConfigMap template:** Render `configMaps.accountDbInit` and `configMaps.orderDbInit` from values.
- **Helpers:** Add reusable templates in `_helpers.tpl` (e.g. common labels, fullname).
- **Chart dependencies:** If we add Redis or another service, we could use Helm subcharts or dependencies.
- **CI/CD integration:** Use Helm in pipelines (e.g. `helm upgrade --install` in GitHub Actions).
- **Multiple environments:** Use different values files (`values-dev.yaml`, `values-prod.yaml`) with `-f` flags.

---

## Summary

We converted from **raw Kubernetes manifests** (`k8s/`) to a **Helm chart** (`gocore/`) that:
- Templates repetitive YAML
- Parameterizes everything via `values.yaml`
- Keeps secrets in `values-secret.yaml` (gitignored)
- Deploys everything with one command
- Avoids conflicts by managing resources only through Helm

The chart is production-ready and follows Helm best practices: no secrets in Git, clear separation of concerns (templates by resource type), and idempotent install/upgrade commands.
