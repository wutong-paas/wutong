# v1.15.0

- wutong-operator(更新 region-config ConfigMap 数据)
- wt-api(新增更新 wutongcluster 接口，在 console 端离线接入集群时调用)

在集群端注入工作集群信息：regionID、reigonCode。以便上报信息到 Console 端时得以识别工作集群。

更新 wutongcluster 自定义资源：

```bash
kubectl apply -f https://ghproxy.com/https://raw.githubusercontent.com/wutong-paas/helm-charts/wutong-operator-1.15.0/charts/wutong-operator/templates/crd/wutong.io_wutongclusters.yaml
```
