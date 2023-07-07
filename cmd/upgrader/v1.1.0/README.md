# upgrade to v1.1.0

**Step 0**: Stop api, worker, eventlog, monitor, node, operator

**Step 1**: Upgrade the database

exec ./upgrade.sql in region database

**Step 2**: Run the upgrader

```bash
# linux amd64
wget https://wutong-paas.obs.cn-east-3.myhuaweicloud.com/upgraders/upgrader-v1.1.0-linux-amd64 -O upgrader-v1.1.0

# linux arm64
wget https://wutong-paas.obs.cn-east-3.myhuaweicloud.com/upgraders/upgrader-v1.1.0-linux-arm64 -O upgrader-v1.1.0

chmod +x upgrader-v1.1.0
./upgrader-v1.1.0 --kubeconfig <your kubeconfig>
```

**Step 3**: Update region component image

Addtional: Update chaos and worker WutongComponent env:

```yaml
  env:
    - name: CI_VERSION
      value: v1.1.0-stable
      # or
      # value: v1.1.0-stable-arm64
```

**Step 4**: Update operator image

**Step 5**: Update cloud-adaptor image
