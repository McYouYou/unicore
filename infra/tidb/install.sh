kubectl create -f https://raw.githubusercontent.com/pingcap/tidb-operator/v1.6.0/manifests/crd.yaml
helm repo add pingcap https://charts.pingcap.org/
kubectl create namespace tidb-admin
helm install tidb-operator pingcap/tidb-operator --namespace=tidb-admin --version=1.6.0 -f values-tidb-operator.yaml --wait

# wait for operator ok

kubectl apply -f cluster.yaml --wait=true --timeout=300s
kubectl apply -f dashboard.yaml
kubectl apply -f monitor.yaml

kubectl create secret generic tidb-secret --from-literal=root=unicore233root --from-literal=unicore=unicore233unicore --namespace=tidb
kubectl apply -f init.yaml
kubectl apply -f svc.yaml

# mysql -P32005 -uunicore -punicore233unicore -h172.18.0.3