kubectl delete -f set.yaml
kubectl delete pvc data-mysql-0 data-mysql-1
kubectl delete -f svc.yaml
kubectl delete -f config.yaml
minikube ssh 'sudo rm -rf /mnt/sda1/pv0' && minikube ssh 'sudo rm -rf /mnt/sda1/pv1'
