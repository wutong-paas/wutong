# v1.14.0

升级内容：

- 集成 wt-api-telepresence-interceptor 组件，建立梧桐管理集群与边端孤立集群的网络隧道，实现梧桐管理集群对边端孤立集群 wt-api 组件的访问。

升级操作：

- 更新 CRD 定义：

```bash
kubectl apply -f https://ghproxy.com/https://raw.githubusercontent.com/wutong-paas/helm-charts/wutong-operator-1.14.0/charts/wutong-operator/templates/crd/wutong.io_wutongclusters.yaml
```

## 全局

约定一个边端孤立集群的 code（edge-isolated-cluster-code），例如： `edge-cluster-01`。

## 梧桐管理集群

1、安装 telepresence traffic manager：

```bash
telepresence helm install
# 有必要的话，通过 --kubeconfig 指定管理集群文件

# 在集群间，经常会遇到 cidr 冲突，可能需要配置允许冲突的子网
telepresence helm install --set client.routing.allowConflictingSubnets='{10.244.0.0/22}'
telepresence helm upgrade --set client.routing.allowConflictingSubnets='{10.244.0.0/22,10.244.4.0/24}'
```

2、部署边端孤立集群 wt-api 的代理组件

```bash
# http
kubectl create deployment <edge-isolated-cluster-code>-wt-api-http -n wt-system --image swr.cn-southwest-2.myhuaweicloud.com/wutong/infinity
kubectl expose deployment <edge-isolated-cluster-code>-wt-api-http -n wt-system --name <edge-isolated-cluster-code>-wt-api-http --port 8888

# ws
kubectl create deployment <edge-isolated-cluster-code>-wt-api-ws -n wt-system --image swr.cn-southwest-2.myhuaweicloud.com/wutong/infinity
kubectl expose deployment <edge-isolated-cluster-code>-wt-api-ws -n wt-system --name <edge-isolated-cluster-code>-wt-api-ws --type NodePort --port 6060
```

3、添加一个边端孤立集群

```json
{
    "regionType": "edge-isloated",
    "clusterCode": "edge-cluster-01",
    "edgeIsloatedClusterConflictingSubnets": [
        "10.244.0.0/22",
        "10.244.4.0/22"
    ]
}
```

这会在 console 端创建一个集群，并且该集群访问 url 为 `http://edge-cluster-01-wt-api.wt-system:8888`，wsurl 则需要额外配置，例如配置 管理集群节点的 IP 和 NodePort `ws://<node-ip>:<node-port>`。

## 边端孤立集群

如果标记集群为边端孤立集群，首先需要在边端孤立集群 `wt-system` 命名空间下创建一个名为 `wt-management-cluster-kubeconfig` 的 secret，内容为梧桐管理集群的 kubeconfig 文件，例如：

```bash
kubectl create secret generic wt-management-cluster-kubeconfig --from-file=kubeconfig=<wt-management-cluster-kubeconfig-file-path> -n wt-system
```

然后，更新 WutongCluster 自定义资源实例，定义 `spec.edgeIsolatedClusterCode` 字段值为边端孤立集群的 code：

```yaml
apiVersion: wutong.io/v1alpha1
kind: WutongCluster
metadata:
  name: wutong-cluster
...
spec:
  edgeIsolatedClusterCode: <edge-isolated-cluster-code>
...
```
