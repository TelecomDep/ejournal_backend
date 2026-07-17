# Preparation
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/configmap.yaml -n ejournal
kubectl apply -f k8s/secret.yaml -n ejournal
kubectl apply -f k8s/tls-secret.yaml -n ejournal

# Postgres + mail
kubectl apply -f k8s/postgres.yaml -n ejournal
kubectl apply -f k8s/mailserver.yaml -n ejournal

# Migration Job
kubectl apply -f k8s/migrate-job.yaml -n ejournal

# Back + Front
kubectl apply -f k8s/backend.yaml -n ejournal
kubectl apply -f k8s/frontend.yaml -n ejournal

# Autoscaler
kubectl apply -f k8s/hpa.yaml -n ejournal
