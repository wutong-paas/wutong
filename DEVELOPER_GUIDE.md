# Developer Guide

本文是 Wutong Region 端组件的开发者指南。

## 组件编译

### 编译环境初始化

由于部分组件 (eventlog) 使用到了 CGO，所以交叉编译还需要额外设置编译环境。参考如下：

基础工具：

- docker
- golang 1.20+

**linux/amd64 环境**：

在 linux 机器 x86 架构开发环境下编译多架构镜像，需要执行以下命令:

```bash
docker run --rm --privileged multiarch/qemu-user-static --reset --persistent yes
```

**macos m1 环境**：

在 M1 芯片 MacOS 机器上编译多架构镜像，需要执行以下命令：

1. 安装依赖

```bash
brew tap messense/macos-cross-toolchains
brew install x86_64-unknown-linux-gnu
brew install aarch64-unknown-linux-gnu
```

2. 添加 PATH

```bash
export PATH=$PATH:/opt/homebrew/Cellar/x86_64-unknown-linux-gnu/11.2.0_1/bin::/opt/homebrew/Cellar/aarch64-unknown-linux-gnu/11.2.0_1/bin
```

### 编译组件

本地编译单个组件并推送：

```bash
make build-api
```

本地编译所有组件镜像并推送：

```bash
make build-all
```
