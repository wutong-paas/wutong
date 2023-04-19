# upgrade to v1.1.0

Step 1: Upgrade the database

exec ./upgrade.sql in region database

Step 2: Run the upgrader

```bash
# linux amd64
wget https://wutong-paas.obs.cn-east-3.myhuaweicloud.com/upgraders/upgrader-v1.1.0-linux-amd64 -O upgrader-v1.1.0

# linux arm64
wget https://wutong-paas.obs.cn-east-3.myhuaweicloud.com/upgraders/upgrader-v1.1.0-linux-arm64 -O upgrader-v1.1.0

chmod +x upgrader-v1.1.0
./upgrader-v1.1.0 --kubeconfig <your kubeconfig>
```
