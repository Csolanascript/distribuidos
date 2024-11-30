kubectl delete pod client   # Eliminar el pod cliente explícitamente

kubectl delete statefulset raft
kubectl delete service raft-service

kubectl create -f statefulset_go.yaml
