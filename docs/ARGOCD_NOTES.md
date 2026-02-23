# How We're Using Argo CD with GoCore

Notes on how Argo CD is used to deploy and manage the GoCore Helm chart in this project.

---

## 1. What Is Argo CD and Why We Use It

- **Argo CD** is a GitOps tool for Kubernetes: it keeps the cluster state in sync with a **source of truth** in Git (or Helm repo).
- **What we do:** The GoCore app is defined in Git (Helm chart in `gocore/`). Argo CD watches the repo and runs `helm template` + apply so the cluster matches what’s in Git.
- **Benefits:** Same config for everyone, audit trail in Git, automatic sync after push, and easy rollback by reverting commits.

---

## 2. How the Application Is Set Up in Argo CD

### 2.1 General

| Field | Value |
|-------|--------|
| **Application Name** | `gocore` (must be **lowercase**; see below) |
| **Project** | Default or your project |
| **Sync Policy** | Automatic (optional: Self Heal, Prune) |

### 2.2 Source

| Field | Value |
|-------|--------|
| **Repository URL** | `https://github.com/sdshah09/GoCore` (or your fork) |
| **Revision** | `HEAD` or a branch (e.g. `main`) |
| **Path** | `gocore` (directory containing `Chart.yaml` and `templates/`) |
| **Type** | **Helm** |

Argo CD treats `gocore` as a Helm chart: it runs `helm template` with that path and applies the rendered manifests.

### 2.3 Destination

| Field | Value |
|-------|--------|
| **Cluster URL** | `https://kubernetes.default.svc` (in-cluster) or your cluster URL |
| **Namespace** | e.g. `default` (where all GoCore resources are created) |

The **Destination Namespace** is passed to Helm as the release namespace so every resource gets `metadata.namespace` set correctly.

### 2.4 Helm-Specific

- **Values files:** If you use a values file in the repo (e.g. `gocore/values.yaml`), configure it in Argo (e.g. “Values Files”: `values.yaml`). For secrets, use a separate file or Parameters (see below).
- **Parameters:** Argo can override Helm values via “Parameters” (each entry is a `--set key=value`). We use this to pass secrets or overrides without storing them in Git (e.g. `secrets.dbCredentials.stringData.postgres-password` from a secret manager or Argo secret).

---

## 3. Requirements We Fixed for Argo CD

### 3.1 Application / Release Name Must Be Lowercase

**Error:**
```text
release name "Gocore": invalid release name, must match regex ^[a-z0-9]([-a-z0-9]*[a-z0-9])?...
```

**Cause:** Helm release names must be lowercase letters, digits, and hyphens only. Argo CD was using the Application name as the release name.

**Fix:** Set **Application Name** to **`gocore`** (all lowercase), not `Gocore`.

### 3.2 Every Resource Must Have a Namespace

**Error:**
```text
Namespace for account-db /v1, Kind=Service is missing.
Namespace for account-db apps/v1, Kind=Deployment is missing.
...
```

**Cause:** Rendered manifests had no `metadata.namespace`. Argo CD needs a namespace to assign resources to the destination.

**Fix:** We added `namespace: {{ .Release.Namespace | default "default" }}` to the `metadata` of every resource in the Helm templates (Deployments, Services, PVCs, Secret, Ingress). When Argo runs `helm template --namespace <destination>`, every resource gets that namespace.

---

## 4. Sync Behavior and Manual Changes

### 4.1 What Argo CD Does on Sync

1. Runs `helm template` with the chart at **Path** and your **Values** / **Parameters**.
2. Compares the rendered manifests to the live cluster.
3. Applies create/update/delete so the cluster matches Git + Helm values.

So the **source of truth** is Git + Helm values, not manual `kubectl` changes.

### 4.2 Why Manual Scaling Doesn’t Stick

If you run:
```bash
kubectl scale deployment account-service --replicas=5
```
and `values.yaml` has `services.account.replicas: 2`, the next Argo CD sync will **revert** the Deployment back to 2 replicas.

**Ways to scale:**

- **Recommended:** Change `services.account.replicas` (or other services) in `gocore/values.yaml`, commit, push. Argo CD syncs and applies the new replica count.
- **Optional:** Configure Argo CD **Ignore Differences** so it ignores `spec.replicas` on Deployments if you want to scale only with `kubectl` (less GitOps-consistent).

### 4.3 Sync Options We Use

| Option | What it does |
|--------|------------------|
| **Automatic** | Argo CD syncs when it detects Git (or Helm) changes. |
| **Self Heal** | If someone changes the cluster manually, Argo reverts it to match Git. |
| **Prune** | Resources that are no longer in the rendered manifests are deleted from the cluster. |

With **Self Heal** on, any manual edit (e.g. `kubectl scale`, `kubectl edit`) will be overwritten on the next sync.

---

## 5. Workflow: Making Changes

1. **Edit the chart or values** in the repo (e.g. `gocore/values.yaml`, `gocore/templates/*.yaml`).
2. **Commit and push** to the branch Argo CD watches (e.g. `main`).
3. Argo CD detects the change and runs a sync (if Auto-Sync is on), or you trigger **Sync** in the UI.
4. The cluster is updated to match the new manifests.

**Secrets:** Don’t put real secrets in Git. Use Argo CD Parameters (or a values file from a secret store) to pass `secrets.dbCredentials.stringData.*` so the Secret is created at sync time without committing passwords.

---

## 6. Summary Table

| Topic | Detail |
|--------|--------|
| **What Argo CD does** | Watches Git repo, runs `helm template` on `gocore/`, applies result to the cluster. |
| **Application name** | `gocore` (lowercase). |
| **Path** | `gocore`. |
| **Source type** | Helm. |
| **Namespace** | Set in Destination; all resources get it via `metadata.namespace` in templates. |
| **Scaling** | Change `replicas` in `values.yaml` and push; don’t rely on `kubectl scale` if Self Heal is on. |
| **Secrets** | Provide via Parameters or a non-committed values file; do not commit secrets to Git. |

This is how we use Argo CD with the GoCore Helm chart: Git + Helm as source of truth, Argo CD keeping the cluster in sync.
