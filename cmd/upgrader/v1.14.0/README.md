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

2、使用 `wtctl` 工具获取边端孤立集群配置信息：

```bash
docker run -it --rm -v /:/rootfs swr.cn-southwest-2.myhuaweicloud.com/wutong/wt-wtctl:v1.14.0 copy
mv /usr/local/bin/wutong-wtctl /usr/local/bin/wtctl
wtctl install
wtctl config
```

3、添加一个边端孤立集

进入 Console 控制台，集群设置 => 添加集群 => 接入已安装平台集群，将步骤 2 中获取到的边端孤立集群配置信息录入。

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

控制逻辑：

1、识别到 `edgeIsolatedClusterCode` 配置，在梧桐管理集群中创建 `<edgeIsolatedClusterCode>-wt-api-agent` Deployment 和 Service，用于代理边端 wt-api 服务；

2、在边端孤立集群中创建 `wt-api-telepresence-interceptor` Deployment，用于连接梧桐管理集群 telepresence traffic manager 并向 `<edgeIsolatedClusterCode>-wt-api-agent` 注入 `traffic-agent` sidecar 容器；

`wt-api-telepresence-interceptor` 将配置健康检查（LivenessProbe），使用 kubectl 工具检测 `<edgeIsolatedClusterCode>-wt-api-agent.wt-system:8888` 服务是否正常连接，如果连接失败，则滚动则重启容器，重启后会重新注入 telepresence traffic-agent。
