# Recover account-db After "invalid checkpoint record" / Corruption

## What happened
PostgreSQL data on the PVC was corrupted (e.g. unclean shutdown). Postgres refuses to start with that data.

## Option A: Delete PVC and let Postgres reinitialize (data loss, clean state)

Run these in order:

```bash
# 1. Scale down so the pod releases the PVC
kubectl scale deployment account-db --replicas=0

# 2. Wait for pod to terminate
kubectl get pods -l app=account-db

# 3. Delete the PVC (this deletes the corrupted data)
kubectl delete pvc account-db-pvc

# 4. Scale back up (new pod + new PVC, Postgres will run init)
kubectl scale deployment account-db --replicas=1

# 5. Wait for pod to be Ready (init can take a minute)
kubectl get pods -l app=account-db -w
```

After this, account-db will have a **fresh** database. Re-run any init scripts (e.g. from ConfigMap) if needed; the postgres image runs scripts in `/docker-entrypoint-initdb.d` only on **first** init (empty data dir).

## Option B: Re-apply deployment and PVC (if PVC was recreated)

```bash
kubectl apply -f k8s/account-db/pvc.yaml
kubectl apply -f k8s/account-db/deployment.yaml
```

## Verify

```bash
kubectl get pods -l app=account-db
kubectl logs -l app=account-db --tail=30
```

You should see "database system is ready to accept connections" and no PANIC.
