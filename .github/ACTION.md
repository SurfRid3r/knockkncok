# GitHub Actions 自动构建和发布

## 工作流说明

本项目包含两个GitHub Actions工作流：

### 1. CI Build (ci.yml)
- **触发条件**: 推送到main/master分支或创建Pull Request
- **功能**: 
  - 构建knockd和kk二进制文件
  - 测试交叉编译功能
  - 确保代码可以正常编译

### 2. Build and Release (build.yml)
- **触发条件**: 
  - 创建v开头的标签 (如v1.0.0)
  - 手动触发
- **功能**:
  - 多平台交叉编译
  - 自动创建GitHub Release
  - 上传编译好的二进制文件

## 支持的平台

| 操作系统 | 架构 | 服务端 | 客户端 | 文件名 |
|---------|------|--------|--------|--------|
| Linux | AMD64 | ✅ | ✅ | `knockd-linux-amd64`, `kk-linux-amd64` |
| Linux | ARM64 | ✅ | ✅ | `knockd-linux-arm64`, `kk-linux-arm64` |
| macOS | AMD64 | ❌ | ✅ | `kk-darwin-amd64` |
| Windows | AMD64 | ❌ | ✅ | `kk-windows-amd64.exe` |

### 说明：
- **服务端 (knockd)**: 仅支持Linux，需要root权限和网络包捕获能力
- **客户端 (kk)**: 支持所有平台，用于发送SPA数据包
- **直接下载**: 用户直接下载二进制文件，无需解压

## 发布流程

1. **创建标签**:
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

2. **自动触发**:
   - GitHub Actions会自动检测到新标签
   - 开始多平台交叉编译
   - 创建GitHub Release
   - 上传所有二进制文件

3. **下载发布**:
   - 访问项目的Releases页面
   - 下载对应平台的二进制文件

## 手动触发

也可以通过GitHub Actions页面手动触发构建流程。

## 注意事项

- 由于knockd使用了pcap库，Windows版本的knockd可能需要额外的依赖
- 建议在Linux系统上运行knockd服务端
- 客户端kk可以在所有支持的平台上运行