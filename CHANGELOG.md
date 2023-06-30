# Changelog

## [0.0.7] - 2023-06-30

### Improve

- `mtu` 默认设置为 `1200` - `80` = `1120`. `1200` 是 webrtc 的默认 `mtu`
- `wgortc` 升级到 `v0.0.11`, 修复了 port 监听不生效的问题

## [0.0.6] - 2023-06-10

### Add

- `config.Addrs` 支持设置多个地址了
- `config.Link` link server 移动到配置文件里, 减少命令行参数

## [0.0.5] - 2023-06-02

### Improve

- 添加版本号, 开始正式启用
