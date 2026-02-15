# Persistent Volumes in Kubernetes – Why, What & How

A learning guide to what we did for persistence (account-db, order-db, product-db) and how to reason about it.

---

## Why Do We Need Persistent Storage?

### The problem: containers are ephemeral

- A **container’s filesystem** lives only as long as the container.
- When a **pod is deleted** (restart, rollout, node failure), the container is recreated and gets a **new, empty filesystem**.
- Anything written inside the container (e.g. database data) is **lost** when the pod goes away.

So:

- **Without persistence:** Every time the Postgres or Elasticsearch pod restarts, all data is gone.
- **With persistence:** We attach a **PersistentVolume** to the pod. Data is written to that volume. When the pod is recreated, the new pod can mount the **same** volume and see the **same** data.

### When to use it

Use persistent storage when the application must **keep data across pod restarts**:

- Databases (PostgreSQL, Elasticsearch, etc.)
- File uploads, caches, or any state that must survive restarts

Do **not** use it for:

- Stateless app binaries, config, or logs you don’t need to keep (emptyDir or no volume is fine).

---

## What Are PersistentVolumes and PersistentVolumeClaims?

Kubernetes splits “durable storage” into two ideas:

### 1. PersistentVolume (PV)

- A **piece of storage** in the cluster (backed by disk, cloud volume, NFS, etc.).
- Usually created by an admin or by a **StorageClass** when someone asks for storage.
- Think of it as: “a disk that exists in the cluster.”

### 2. PersistentVolumeClaim (PVC)

- A **request for storage** by a user/workload: “I need X Gi of storage with Y access mode.”
- The cluster finds (or creates) a **PersistentVolume** that matches and **binds** it to the claim.
- Pods don’t use PVs directly; they use **PVCs**. The pod says: “mount the volume from this PVC.”

### How they connect

```
Pod (e.g. postgres)
  → volumeMounts: postgres-storage at /var/lib/postgresql/data
  → volumes: postgres-storage from PVC "account-db-pvc"
       → PVC "account-db-pvc" (request: 500Mi, ReadWriteOnce)
            → bound to PersistentVolume (actual disk)
```

So: **Pod uses PVC → PVC is bound to PV → PV is the real storage.** Data written to the mount path is written to the PV and survives pod restarts.

---

## Access Modes (Why ReadWriteOnce?)

A PVC specifies an **accessMode**:

| Mode             | Meaning                          | Typical use        |
|------------------|-----------------------------------|--------------------|
| **ReadWriteOnce**| One node can mount read-write     | Single-pod databases (our case) |
| ReadOnlyMany     | Many nodes, read-only             | Shared config/data |
| ReadWriteMany    | Many nodes, read-write            | NFS, shared FS     |

We use **ReadWriteOnce** because:

- Each DB runs as **one pod** (replicas: 1).
- Only that pod needs read-write access to its data.
- ReadWriteOnce is widely supported (local disk, cloud disks, etc.).

---

## What We Did in This Project

We added **persistent storage for all three data stores**:

| Component   | Role              | PVC name        | Size  | Mount path                          |
|------------|-------------------|-----------------|-------|-------------------------------------|
| account-db | PostgreSQL        | account-db-pvc  | 500Mi | /var/lib/postgresql/data            |
| order-db   | PostgreSQL        | order-db-pvc    | 500Mi | /var/lib/postgresql/data            |
| product-db | Elasticsearch     | product-db-pvc  | 5Gi   | /usr/share/elasticsearch/data       |

Why different sizes?

- Postgres: 500Mi is enough for dev/small datasets.
- Elasticsearch: indices and shards can grow; 5Gi gives headroom.

---

## How It’s Wired (Step by Step)

### Step 1: Define the claim (PVC)

We create a **PersistentVolumeClaim** so the cluster provisions (or binds) storage.

**Example – account-db** (`k8s/account-db/pvc.yaml`):

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: account-db-pvc
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 500Mi
```

- **name:** Used by the Deployment when it asks for this volume.
- **accessModes: ReadWriteOnce:** One node, read-write.
- **resources.requests.storage:** Minimum size we want.

Same idea for **order-db** (500Mi) and **product-db** (5Gi); only names and sizes differ.

### Step 2: Use the claim in the Deployment

In the **Deployment** we do two things:

1. **Declare a volume** that comes from the PVC.
2. **Mount that volume** in the container at the path the app expects.

**Example – account-db** (snippet):

```yaml
spec:
  template:
    spec:
      containers:
      - name: postgres
        volumeMounts:
        - name: postgres-storage
          mountPath: /var/lib/postgresql/data
      volumes:
      - name: postgres-storage
        persistentVolumeClaim:
          claimName: account-db-pvc
```

- **volumes:** “There is a volume named `postgres-storage` backed by the PVC `account-db-pvc`.”
- **volumeMounts:** “In the container, mount that volume at `/var/lib/postgresql/data`.”

PostgreSQL writes all its data under that path, so it now writes to the PV. When the pod is recreated, the new pod mounts the same PVC (same PV), so data persists.

**Product-db** is the same idea, different path:

- **Volume:** `elasticsearch-data` from PVC `product-db-pvc`.
- **Mount:** `/usr/share/elasticsearch/data` (where Elasticsearch stores indices).

---

## Why We Added a preStop Hook (PostgreSQL)

We saw **data directory corruption** when the Postgres pod was killed abruptly (invalid checkpoint record). So we added a **lifecycle preStop** hook to shut Postgres down cleanly before the container exits:

```yaml
lifecycle:
  preStop:
    exec:
      command:
      - sh
      - -c
      - pg_ctl stop -D /var/lib/postgresql/data -m fast -w || true
```

- **preStop:** Runs when the pod is terminating (e.g. delete, rollout).
- **pg_ctl stop -m fast:** Tells Postgres to stop and flush a consistent checkpoint.
- **|| true:** So the pod still terminates even if pg_ctl fails (e.g. already stopped).

This reduces the chance of corruption on the next start. We use it for **account-db** and **order-db**; Elasticsearch is less sensitive in the same way, but you could add a similar hook if needed.

---

## How to Test That Data Persists

1. **Insert data** into the DB (real tables/index).
2. **Delete the pod** (e.g. `kubectl delete pod -l app=order-db`).
3. **Wait** for the new pod to be Ready (same Deployment, so it recreates the pod).
4. **Check** that the data is still there (same PVC is mounted by the new pod).

Example for **order-db** (one row in `orders` and `order_products`):

```bash
ORDER_DB_POD=$(kubectl get pod -l app=order-db -o jsonpath='{.items[0].metadata.name}')
kubectl exec -it $ORDER_DB_POD -- psql -U postgres -d gocore -c "
  INSERT INTO orders (id, created_at, account_id, total_price)
  VALUES ('0testorder000000000000001', NOW(), '0testaccount00000000000001', 99.99);
  INSERT INTO order_products (order_id, product_id, quantity)
  VALUES ('0testorder000000000000001', '0testproduct0000000000001', 2);
  SELECT * FROM orders; SELECT * FROM order_products;
"
kubectl delete pod -l app=order-db
kubectl get pods -l app=order-db -w
# After new pod is Ready:
ORDER_DB_POD=$(kubectl get pod -l app=order-db -o jsonpath='{.items[0].metadata.name}')
kubectl exec -it $ORDER_DB_POD -- psql -U postgres -d gocore -c "SELECT * FROM orders; SELECT * FROM order_products;"
```

Example for **product-db** (one document in `products` index):

```bash
PRODUCT_DB_POD=$(kubectl get pod -l app=product-db -o jsonpath='{.items[0].metadata.name}')
kubectl exec -it $PRODUCT_DB_POD -- curl -s -X PUT "localhost:9200/products/_doc/persistence-test-1" \
  -H "Content-Type: application/json" -d '{"name":"Persistence Test","description":"PVC test","price":49.99}'
kubectl delete pod -l app=product-db
kubectl get pods -l app=product-db -w
# After new pod is Ready:
PRODUCT_DB_POD=$(kubectl get pod -l app=product-db -o jsonpath='{.items[0].metadata.name}')
kubectl exec -it $PRODUCT_DB_POD -- curl -s "localhost:9200/products/_doc/persistence-test-1?pretty"
```

If the rows/document are still there after the new pod is up, persistence is working.

---

## Important Behaviors to Know

1. **PVC is bound to one PV.** Deleting the pod does **not** delete the PVC or the PV; the new pod gets the same claim and same data.
2. **Deleting the PVC** can delete the underlying storage (depending on StorageClass reclaim policy). Only do that when you intend to wipe that data (e.g. recovery from corruption).
3. **ReadWriteOnce** means the volume can only be mounted read-write by one node. So only one pod (one replica) can use it for our DBs. Scaling to multiple replicas would need a different storage or pattern (e.g. read replicas with their own PVCs).
4. **First-time init:** Postgres runs scripts in `/docker-entrypoint-initdb.d` only when the data directory is **empty**. So the first time you use a new PVC, init runs; after that, existing data is used and init is skipped.

---

## If Something Goes Wrong (Corruption / Recovery)

- **Postgres:** If you see “invalid checkpoint record” or similar, the data on the PVC may be corrupted (e.g. from an old crash without preStop). Recovery is to create a **new** empty volume and re-init:
  - Scale the Deployment to 0.
  - Delete the PVC (and PV if it was created and you want a clean slate).
  - Recreate the PVC, scale the Deployment back to 1, and let init run again (see `docs/ACCOUNT_DB_RECOVERY.md` for exact steps).
- **Elasticsearch:** Same idea: if the data dir is broken, you can delete the PVC and let a new pod create a fresh index (data loss but cluster healthy again).

---

## Summary Table (What We Did)

| Topic              | What we did |
|--------------------|-------------|
| **Why**            | So DB data survives pod restarts (containers are ephemeral). |
| **What**           | PVCs (claims) + volumes in Deployments mounting at the app’s data path. |
| **Where**          | account-db, order-db (Postgres), product-db (Elasticsearch). |
| **How**            | 1) Create PVC YAML. 2) In Deployment: add volume from PVC, mount in container at data path. 3) For Postgres, add preStop for graceful shutdown. |
| **Check**          | Insert data → delete pod → new pod comes up → same data visible. |

This is the **why**, **what**, and **how** of persistent volumes in this project; you can reuse the same pattern for other stateful workloads.
