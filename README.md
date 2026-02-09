# 可售卖视频 API 服务

这个仓库提供一个可直接售卖的 API 基础实现，具备：

- API Key 鉴权（`Authorization: Bearer` 或 `X-API-Key`）
- 按 Key 的限流控制（默认每分钟 30 次）
- 访问日志（`data/usage.log`）
- 健康检查接口

## 快速启动

```bash
python3 api_server.py
```

服务默认运行在 `http://localhost:8080`。

## API 说明

### 健康检查

```http
GET /health
```

返回：

```json
{
  "status": "ok",
  "timestamp": 1700000000
}
```

### 随机视频接口

```http
GET /v1/video/random
```

请求头：

```
Authorization: Bearer demo-basic-key
```

或：

```
X-API-Key: demo-basic-key
```

示例响应：

```json
{
  "url": "https://api.jkyai.top/API/jxhssp.php",
  "plan": "basic",
  "cache_ttl": 0,
  "timestamp": 1700000000
}
```

## API Key 配置

编辑 `data/api_keys.json`：

```json
{
  "keys": [
    {
      "key": "demo-basic-key",
      "plan": "basic",
      "rate_limit_per_minute": 30
    }
  ]
}
```

## 作为可售卖 API 的建议

- 将 `data/api_keys.json` 接入数据库或后台管理系统。
- 结合网关或反向代理开启 HTTPS 与更强限流策略。
- 使用独立监控/计费系统读取 `data/usage.log` 进行计费。
- 对上游资源进行可用性监测与缓存。
