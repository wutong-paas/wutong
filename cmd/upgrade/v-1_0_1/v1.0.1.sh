# 1、获取所有由 wutong 创建的命名空间
namespaces = $(kubectl get namespace -o name --selector=app.kubernetes.io/managed-by=wutong)

# 2、删除所有由 wutong 创建的命名空间
for ns in $namespaces; do
    # 2.1、获取到一个 pod
    kubectl label pod --namespace $ns
done
