# v1.13.0

## 前提条件

- kubevirt 部署;
- 集群节点配置 `wutong.io/vm-schedulable: "true"` 标签，标识该节点允许虚拟机调度;
- 集群节点添加虚拟机调度标签，虚拟机配置时可选。

## 虚拟机 virt-vnc 可视化界面

部署 `wutong-go-console`

一体机前端配置

1. nginx 配置

```nginx
  location ~ ^/console/virt-vnc {
      proxy_connect_timeout 1800;
      proxy_send_timeout 1800;
      proxy_read_timeout 1800;
      send_timeout 1800;
      proxy_http_version 1.1; # 确保使用 HTTP/1.1 协议，因为 WebSocket 基于此
      proxy_set_header Upgrade $http_upgrade; # 转发 Upgrade 头部
      proxy_set_header Connection "upgrade"; # 转发 Connection 头部
      proxy_set_header Host $host; # 转发 Host 头部
      proxy_set_header X-Real-IP $remote_addr; # 转发真实 IP 地址
      proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for; # 转发 X-Forwarded-For 头部
      proxy_set_header X-Forwarded-Proto $scheme; # 转发协议类型
      proxy_pass http://wutong-go-console.wutong-console:8000; # 视部署情况修改
  }
```

2. 网关配置，开启 websocket 服务
