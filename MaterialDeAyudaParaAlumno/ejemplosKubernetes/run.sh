#!/bin/bash

#/usr/local/bin/cliente 10.244.4.2:7000 10.244.1.2:6000 loquesea
# Paso 1: Eliminar el clúster existente
#echo "Eliminando el clúster"
./delete_cluster.sh &>/dev/null

# Paso 2: Crear un nuevo clúster
#echo "Creando el clúster"
./kind-with-registry.sh

# Paso 3: Eliminar binarios antiguos
rm Dockerfiles/servidor/servidor
rm Dockerfiles/cliente/cliente

# Paso 4: Compilar los archivos de Go
echo "Compilando los ficheros Golang"
cd raft/cmd/srvraft
CGO_ENABLED=0 go build -o ../../../Dockerfiles/servidor/servidor .

cd ../../pkg/cltraft
CGO_ENABLED=0 go build -o ../../../Dockerfiles/cliente/cliente .

# Paso 5: Crear imágenes Docker
echo "Creando las imágenes Docker"
cd ../../../Dockerfiles/servidor
docker build . -t localhost:5001/servidor:latest
docker push localhost:5001/servidor:latest

cd ../cliente
docker build . -t localhost:5001/cliente:latest
docker push localhost:5001/cliente:latest

cd ../..

# Paso 6: Lanzar los recursos de Kubernetes
echo "Lanzando Kubernetes"
kubectl delete statefulset raft &>/dev/null
kubectl delete pod client &>/dev/null
kubectl delete service raft-service &>/dev/null
kubectl create -f statefulset_go.yaml
#kubectl exec client -ti -- sh
#find / -name cliente
#kubectl exec client -ti -- sh
#/usr/local/bin/cliente
