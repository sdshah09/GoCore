# Helm Chart for GoCore – Complete Notes

A learning and reference guide for the **gocore** Helm chart: what it is, what’s in it, how to use it, and every important detail.

---

## 1. What Is This Helm Chart?

- **Helm** is the “package manager” for Kubernetes: you define a **chart** (templates + default values), and Helm installs/upgrades a **release** (all the K8s resources) in one go.
- The **gocore** chart deploys the full GoCore stack: 4 microservices (account, order, product, graphql), 3 databases (account-db, order-db, product-db), Secret, PVCs, ConfigMaps, Services, and Ingress.
- **One command** (`helm install` or `helm upgrade --install`) creates or updates everything; no need to `kubectl apply` many folders.

---

## 2. Chart Layout (What Files Exist and What They Do)

```
gocore/
├── Chart.yaml              # Chart metadata (name, version, appVersion)
├── values.yaml             # Default config (no secrets); committed to Git
├── values-secret.yaml      # Secrets only; gitignored, not committed
├── values-secret.yaml.example   # Template for secrets; committed
└── templates/              # Helm templates (mix of Go templating + YAML)
    ├── _helpers.tpl        # Optional; shared template helpers (currently empty)
    ├── deployment.yaml     # Deployments for account, order, product, graphql
    ├── db-deployment.yaml  # PVCs + Deployments for account-db, order-db, product-db
    ├── service.yaml        # Services for the 4 microservices
    ├── dbService.yaml      # Services for the 3 DBs
    ├── secret.yaml         # Secret db-credentials (from values-secret)
    └── ingress.yaml        # Ingress (e.g. / → graphql:8080)
```

### 2.1 Chart.yaml

- **Purpose:** Chart identity and version.
- **Important fields:** `name`, `version`, `appVersion`, `type: application`.
- **When to change:** Bump `version` when you change the chart; bump `appVersion` when the app images change.

### 2.2 values.yaml (committed)

- **Purpose:** All non-secret configuration.
- **Key sections:**
  - **global:** `imagePullPolicy`, `dbCredentialsSecret` name.
  - **services:** account, order, product, graphql (replicas, image, ports, env, probes, resources). Used by `deployment.yaml` and `service.yaml`.
  - **dbServices:** List used by `dbService.yaml` (ref, name, selector, ports).
  - **accountDb, orderDb, productDb:** Full DB config (image, probes, PVC, lifecycle, resources). Used by `db-deployment.yaml` and `dbService.yaml`.
  - **configMaps:** account-db-init, order-db-init (init SQL). Rendered elsewhere if you add a configmap template.
  - **ingress:** enabled, name, className, rules (path / → graphql:8080).
- **Secrets:** Only structure/comments here; no real credentials (so safe to push to GitHub).

### 2.3 values-secret.yaml (gitignored)

- **Purpose:** Hold DB credentials so they are **not** in Git.
- **Content:** `secrets.dbCredentials.stringData` (e.g. `postgres-user`, `postgres-password`).
- **Usage:** Always pass with `-f gocore/values-secret.yaml` when installing/upgrading.
- **Created from:** Copy `values-secret.yaml.example` to `values-secret.yaml` and edit.

### 2.4 values-secret.yaml.example (committed)

- **Purpose:** Show the format for the secret values file.
- **Usage:** `cp gocore/values-secret.yaml.example gocore/values-secret.yaml` then edit the password.

### 2.5 templates/deployment.yaml

- **Renders:** One Deployment per entry in `.Values.services` (account, order, product, graphql) when enabled.
- **Uses:** name, replicas, image, ports (grpc/app + health), probes, resources, env. Env keys ending in `_SECRET` become `secretKeyRef` from `db-credentials`.
- **Ports:** Main port from `ports.grpc` or `ports.app`; health port named `liveness-port` (8080).

### 2.6 templates/db-deployment.yaml

- **Renders:** For each of accountDb, orderDb, productDb (when enabled):
  - One **PersistentVolumeClaim** (name, size, accessModes).
  - One **Deployment** (Postgres with exec probes + preStop + init ConfigMap + data volume, or Elasticsearch with httpGet probes + data volume).
- **Branching:** If `initConfigMap` is set → Postgres style; else → Elasticsearch style.

### 2.7 templates/service.yaml

- **Renders:** One Service per entry in `.Values.services` (same four microservices).
- **Ports:** If `service` block exists (graphql) → single port; else → grpc + health from `ports`.

### 2.8 templates/dbService.yaml

- **Renders:** One Service per entry in `.Values.dbServices` (account-db, order-db, product-db). Uses ref + enabled from the corresponding DB value (accountDb, orderDb, productDb).

### 2.9 templates/secret.yaml

- **Renders:** One Secret `db-credentials` **only if** `secrets.dbCredentials` has `stringData` or `data`.
- **Data:** From `values-secret.yaml` (or `--set`). Prefer `stringData` (plain text) so nothing is committed.

### 2.10 templates/ingress.yaml

- **Renders:** One Ingress when `ingress.enabled` is true. Uses `ingress.name`, `ingress.className`, and `ingress.rules` (path / → backend service name + port).

### 2.11 templates/_helpers.tpl

- **Purpose:** Reusable named templates (e.g. labels, fullname). Filenames starting with `_` are not rendered as manifests.
- **Current state:** Empty; can be used later for shared snippets.

---

## 3. Values Structure (Quick Reference)

| Key | Used by | Purpose |
|-----|---------|--------|
| global.imagePullPolicy | deployments, db-deployment | Container image pull policy |
| global.dbCredentialsSecret | deployment, db-deployment, secret | Secret name for DB credentials |
| services.* | deployment.yaml, service.yaml | Microservice config (account, order, product, graphql) |
| dbServices | dbService.yaml | List of DB service definitions (name, selector, ports) |
| accountDb, orderDb, productDb | db-deployment.yaml, dbService (via ref) | DB deployment + PVC + probes |
| secrets.dbCredentials | secret.yaml | Secret name and stringData/data |
| ingress | ingress.yaml | Ingress name, class, rules |
| configMaps | (optional configmap template) | Init SQL for Postgres DBs |

---

## 4. Helm Commands – What and Why

### 4.1 Install (first-time deploy)

```bash
helm install gocore ./gocore -f gocore/values.yaml -f gocore/values-secret.yaml
```

- **What:** Creates release `gocore` from chart `./gocore`, merging values from the two files (later file overrides).
- **Why:** Deploy the whole stack once. Use the correct filename: `values-secret.yaml` (not `values-secret.yamll`).

### 4.2 Upgrade (after changing values or templates)

```bash
helm upgrade gocore ./gocore -f gocore/values.yaml -f gocore/values-secret.yaml
```

- **What:** Updates the existing release with new/changed templates or values.
- **Why:** Apply config or chart changes without uninstalling.

### 4.3 Upgrade or install (idempotent)

```bash
helm upgrade --install gocore ./gocore -f gocore/values.yaml -f gocore/values-secret.yaml
```

- **What:** If `gocore` does not exist → install; if it exists → upgrade.
- **Why:** Same command for first deploy and every update (scripts, CI).

### 4.4 Template (dry run – no cluster)

```bash
helm template gocore ./gocore -f gocore/values.yaml -f gocore/values-secret.yaml
```

- **What:** Renders all manifests to stdout; does not talk to the cluster.
- **Why:** Debug what would be applied; feed into GitOps or `kubectl apply --dry-run`.

### 4.5 Uninstall (remove release)

```bash
helm uninstall gocore -n default
```

- **What:** Deletes the release and most resources Helm created. PVCs may be kept by default (resource policy).
- **Why:** Tear down the app; clean before a fresh install.

### 4.6 List releases

```bash
helm list -n default
```

- **What:** Shows releases (name, namespace, revision, status, chart).
- **Why:** Confirm install and revision.

### 4.7 Release status and resources

```bash
helm status gocore -n default
```

- **What:** Shows release status and the list of resources (Pods, Services, PVCs, etc.).
- **Why:** Quick overview of what was deployed.

### 4.8 Override a single value

```bash
helm upgrade --install gocore ./gocore -f gocore/values.yaml -f gocore/values-secret.yaml --set services.account.replicas=3
```

- **What:** Merges `-f` files, then overrides with `--set`. Multiple `--set key=value` allowed.
- **Why:** One-off overrides without editing files (e.g. replicas, or a secret from env).

---

## 5. Clean Install Procedure (Avoiding “Already Exists” and Conflicts)

If you previously applied the same resources with `kubectl apply -f k8s/`, those resources are **not** owned by Helm. Helm will then fail with “cannot be imported” or “conflict with kubectl-client-side-apply”. Do **one** of the following.

### 5.1 Recommended: Full clean, then Helm only

1. Uninstall the release (if it exists):
   ```bash
   helm uninstall gocore -n default --ignore-not-found
   ```
2. Delete the conflicting resources that were created by `kubectl` (so Helm can create them itself):
   ```bash
   kubectl delete deployment account-db order-db product-db account-service order-service product-service graphql -n default --ignore-not-found
   kubectl delete ingress gocore-ingress -n default --ignore-not-found
   kubectl delete pvc account-db-pvc order-db-pvc product-db-pvc -n default --ignore-not-found
   kubectl delete secret db-credentials -n default --ignore-not-found
   kubectl delete svc account-db order-db product-db account-service order-service product-service graphql -n default --ignore-not-found
   ```
3. Install with Helm (fix typo: use `values-secret.yaml` not `values-secret.yamll`):
   ```bash
   helm install gocore ./gocore -f gocore/values.yaml -f gocore/values-secret.yaml
   ```

**Important:** After a clean install, manage this app **only with Helm** in that namespace. Do not mix `kubectl apply -f k8s/` for the same resources.

### 5.2 If PVCs are “Terminating” and stuck

- Wait for them to disappear, or check for finalizers and fix (advanced). Then rerun the delete and install.

### 5.3 Do not “adopt” by hand

- Adding Helm labels/annotations to existing resources (e.g. PVCs, Deployments) still leads to **apply conflicts** (e.g. `imagePullPolicy`, `resources`, `spec.rules`) because the object was first created by kubectl. Prefer deleting and letting Helm recreate.

---

## 6. Small Details and Gotchas

| Item | Detail |
|------|--------|
| **Typo** | Use `values-secret.yaml` (one `l`). `values-secret.yamll` will fail or use wrong file. |
| **Secrets in Git** | Never put real credentials in `values.yaml`. Use `values-secret.yaml` (gitignored) or `--set`. |
| **Option B for secrets** | Keep a local `values-secret.yaml`; copy from `values-secret.yaml.example` on a new clone. |
| **.gitignore** | Patterns like `*-secret*.yaml` and `values-*.local.yaml` keep secret files from being committed. |
| **ConfigMaps** | account-db-init and order-db-init are in values; if you add a configmap template, render them from `configMaps`. |
| **DB credentials key names** | In values, env keys are `DB_USER_SECRET` and `DB_PASSWORD_SECRET`; the template turns them into env vars `DB_USER` / `DB_PASSWORD` from the secret keys (e.g. `postgres-user`, `postgres-password`). |
| **graphql Service** | Single port 8080; in values it uses `service.port` and `service.targetPort` under `services.graphql`. |
| **product-db** | Elasticsearch; no `initConfigMap`; httpGet probes; single data volume. |
| **account-db / order-db** | Postgres; have `initConfigMap`, `lifecycle.preStop`, exec probes, two volumes (init script + data). |

---

## 7. Summary Table

| Goal | Command or action |
|------|-------------------|
| First-time deploy | `helm install gocore ./gocore -f gocore/values.yaml -f gocore/values-secret.yaml` |
| Update after changes | `helm upgrade gocore ./gocore -f gocore/values.yaml -f gocore/values-secret.yaml` |
| Install or upgrade | `helm upgrade --install gocore ./gocore -f gocore/values.yaml -f gocore/values-secret.yaml` |
| See rendered YAML | `helm template gocore ./gocore -f gocore/values.yaml -f gocore/values-secret.yaml` |
| Remove release | `helm uninstall gocore -n default` |
| List releases | `helm list -n default` |
| Release status + resources | `helm status gocore -n default` |
| New machine / clone | Copy `values-secret.yaml.example` to `values-secret.yaml`, edit password, then install/upgrade with `-f gocore/values-secret.yaml` |
| Avoid conflicts | Don’t mix `kubectl apply -f k8s/` with Helm for the same resources; do a clean delete of those resources then use only Helm. |

---

## 8. File-to-Resource Mapping

| Template file | Kubernetes resources produced |
|---------------|-------------------------------|
| deployment.yaml | Deployment: account-service, order-service, product-service, graphql |
| db-deployment.yaml | PVC: account-db-pvc, order-db-pvc, product-db-pvc; Deployment: account-db, order-db, product-db |
| service.yaml | Service: account-service, order-service, product-service, graphql |
| dbService.yaml | Service: account-db, order-db, product-db |
| secret.yaml | Secret: db-credentials |
| ingress.yaml | Ingress: gocore-ingress |

This document is the single place for “how to create, what files, what each does, how to use, commands, and every small thing” for the GoCore Helm chart. You can paste it into Notion or keep it in the repo as `docs/HELM_CHART_NOTES.md`.
