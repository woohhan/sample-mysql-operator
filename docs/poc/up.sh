minikube ssh 'sudo mkdir -p /mnt/sda1/pv0' && minikube ssh 'sudo mkdir -p /mnt/sda1/pv1'
kubectl apply -f config.yaml
kubectl apply -f svc.yaml
kubectl apply -f set.yaml
kubectl get pods -l app=mysql --watch
