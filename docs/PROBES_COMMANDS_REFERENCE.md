# Probes & Health – Command Reference

All commands assume you’re in the project root and `kubectl` is pointed at your cluster (e.g. kind-gocore). Replace `<pod-name>` or labels with your actual pod names/labels where needed.

---

## 1. Apply / Update Manifests (Probes Live in Deployments)

```bash
# Apply all app deployments (readiness + liveness + startup probes)
kubectl apply -f k8s/account/deployment.yaml
kubectl apply -f k8s/order/deployment.yaml
kubectl apply -f k8s/product/deployment.yaml
kubectl apply -f k8s/graphql/deployment.yaml

# Apply all DB deployments (DB probes)
kubectl apply -f k8s/account-db/deployment.yaml
kubectl apply -f k8s/order-db/deployment.yaml
kubectl apply -f k8s/product-db/deployment.yaml

# Apply services (so port 8080 is exposed for health)
kubectl apply -f k8s/account/service.yaml
kubectl apply -f k8s/order/service.yaml
kubectl apply -f k8s/product/service.yaml
kubectl apply -f k8s/graphql/service.yaml

# Apply ingress (graphql at / so localhost/health and localhost/ready work)
kubectl apply -f k8s/ingress/ingress.yaml
```

---

## 2. Check Pods and Probes (Readiness / Liveness)

```bash
# List all pods; READY shows readiness (e.g. 1/1 = ready, 0/1 = not ready)
# RESTARTS often indicate liveness probe failures
kubectl get pods

# Same, with node and IP
kubectl get pods -o wide

# Pods with READY, RESTARTS, and image (RESTARTS = liveness restarts)
kubectl get pods -o custom-columns=NAME:.metadata.name,STATUS:.status.phase,READY:.status.containerStatuses[0].ready,RESTARTS:.status.containerStatuses[0].restartCount,IMAGE:.spec.containers[0].image

# List only pods for a given app (by label)
kubectl get pods -l app=account
kubectl get pods -l app=order
kubectl get pods -l app=product
kubectl get pods -l app=graphql
kubectl get pods -l app=account-db
kubectl get pods -l app=order-db
kubectl get pods -l app=product-db
```

```bash
# See why a pod is not ready (readiness) or restarted (liveness)
# Shows Conditions (Ready True/False), Events (e.g. probe failures)
kubectl describe pod <pod-name>

# Example for one of the product pods
kubectl describe pod -l app=product
```

```bash
# Recent cluster events (probe failures, kills, scheduling)
kubectl get events --sort-by='.lastTimestamp'

# Last N lines
kubectl get events --sort-by='.lastTimestamp' | tail -50
```

```bash
# Which pods are NOT ready (readiness failed)
kubectl get pods -o jsonpath='{range .items[?(@.status.conditions[?(@.type=="Ready")].status!="True")]}{.metadata.name}{"\t"}{.status.conditions[?(@.type=="Ready")].reason}{"\n"}{end}'
```

```bash
# Deployment status (ready replicas vs desired)
kubectl get deployments
kubectl get deployments -o custom-columns=NAME:.metadata.name,READY:.status.readyReplicas/DESIRED:.spec.replicas,IMAGE:.spec.template.spec.containers[0].image
```

---

## 3. Rollout (Restart Deployments – Probes Control Traffic During Rollout)

```bash
# Restart one deployment (new pods; readiness/liveness apply)
kubectl rollout restart deployment account-service
kubectl rollout restart deployment order-service
kubectl rollout restart deployment product-service
kubectl rollout restart deployment graphql
kubectl rollout restart deployment account-db
kubectl rollout restart deployment order-db
kubectl rollout restart deployment product-db

# Restart all app + DB deployments
kubectl rollout restart deployment account-db order-db product-db account-service order-service product-service graphql
```

```bash
# Wait until rollout is done (uses readiness)
kubectl rollout status deployment account-service
kubectl rollout status deployment product-service
# etc.
```

```bash
# Rollout history (last few revisions)
kubectl rollout history deployment product-service
```

---

## 4. Hitting Health Endpoints (What the Probes Hit)

### 4.1 Via Ingress (GraphQL only – port 80)

```bash
# Probes for graphql use /health and /ready on port 8080; ingress sends / to graphql:8080
# So these hit the same handlers the probes use
curl http://localhost/health
curl http://localhost/ready
```

### 4.2 Via Port-Forward (Same as Probes: Service → Pod :8080)

```bash
# Account – forward local 9081 to account-service:8080 (health port)
kubectl port-forward svc/account-service 9081:8080
# Then: curl http://localhost:9081/health  and  curl http://localhost:9081/ready

# Order
kubectl port-forward svc/order-service 9083:8080
# curl http://localhost:9083/health  and  curl http://localhost:9083/ready

# Product
kubectl port-forward svc/product-service 9082:8080
# curl http://localhost:9082/health  and  curl http://localhost:9082/ready

# GraphQL (same port as app)
kubectl port-forward svc/graphql 9080:8080
# curl http://localhost:9080/health  and  curl http://localhost:9080/ready
```

### 4.3 From Inside the Cluster (Like Kubelet Hitting Probes)

```bash
# Run a temporary pod and call health endpoints (no port-forward)
kubectl run curl --rm -it --restart=Never --image=curlimages/curl -- sh

# Inside the pod (same host:port the probes use):
curl -s http://account-service:8080/health
curl -s http://account-service:8080/ready
curl -s http://order-service:8080/health
curl -s http://order-service:8080/ready
curl -s http://product-service:8080/health
curl -s http://product-service:8080/ready
curl -s http://graphql:8080/health
curl -s http://graphql:8080/ready
exit
```

```bash
# Busybox has wget, not curl
kubectl run test --rm -it --image=busybox --restart=Never -- sh
wget -qO- http://account-service:8080/health
wget -qO- http://account-service:8080/ready
exit
```

---

## 5. Logs (Probes Don’t Log; App Logs Show the Requests)

```bash
# Logs from all pods of an app (you’ll see GET /health, /ready from probes)
kubectl logs -l app=account --tail=100
kubectl logs -l app=order --tail=100
kubectl logs -l app=product --tail=100
kubectl logs -l app=graphql --tail=100

# Follow logs
kubectl logs -l app=product -f --tail=50

# Previous container (after a liveness restart)
kubectl logs <pod-name> --previous
```

---

## 6. Image Versions (Probes Use Whatever Image the Pod Runs)

```bash
# Image per deployment
kubectl get deployments -o custom-columns=NAME:.metadata.name,IMAGE:.spec.template.spec.containers[0].image,READY:.status.readyReplicas/DESIRED:.spec.replicas

# Image per pod
kubectl get pods -o custom-columns=NAME:.metadata.name,IMAGE:.spec.containers[0].image,RESTARTS:.status.containerStatuses[0].restartCount,READY:.status.containerStatuses[0].ready
```

---

## 7. Free a Port (If Port-Forward Fails: “address already in use”)

```bash
# Find process on port 9081
lsof -i :9081

# Kill it
lsof -ti :9081 | xargs kill -9

# Then retry port-forward
kubectl port-forward svc/account-service 9081:8080
```

---

## 8. Build and Load Images (After Changing /health or /ready Code)

```bash
cd /Users/shaswatshah/Desktop/projects/GoCore

# Build (e.g. after fixing /ready or probe-related code)
docker build -f account/app.dockerfile -t account-service:v3 .
docker build -f order/app.dockerfile -t order-service:v3 .
docker build -f product/app.dockerfile -t product-service:v3 .
docker build -f graphql/app.dockerfile -t graphql-service:v2 .

# Load into kind so deployments can use them
kind load docker-image account-service:v3 order-service:v3 product-service:v3 graphql-service:v2 --name gocore
```

Then update deployment image tags if needed and:

```bash
kubectl apply -f k8s/account/deployment.yaml
kubectl apply -f k8s/order/deployment.yaml
kubectl apply -f k8s/product/deployment.yaml
kubectl apply -f k8s/graphql/deployment.yaml
# Or: kubectl rollout restart deployment ...
```

---

## 9. Quick “Probes and Health” Checklist

```bash
# 1) All pods running and ready
kubectl get pods

# 2) Recent events (probe failures, restarts)
kubectl get events --sort-by='.lastTimestamp' | tail -20

# 3) GraphQL health via ingress
curl -s http://localhost/health && echo ""
curl -s http://localhost/ready && echo ""

# 4) One app via port-forward (e.g. product)
kubectl port-forward svc/product-service 9082:8080 &
sleep 2
curl -s http://localhost:9082/health && echo ""
curl -s http://localhost:9082/ready && echo ""
kill %1 2>/dev/null
```

---

**Summary:** Probes are defined in **Deployments** (readinessProbe, livenessProbe, startupProbe). These commands let you apply them, inspect their effect (pod READY, RESTARTS, events), and call the same **/health** and **/ready** endpoints that the probes use (via ingress, port-forward, or from inside the cluster).
