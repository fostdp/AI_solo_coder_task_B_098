# 古代烽火台视线分析与通信网络可靠性仿真系统

## 项目简介

本系统是一套用于研究汉代河西走廊烽火台体系的全栈仿真应用。通过数字化建模和仿真技术，还原古代军事通信网络的运作机制，分析视线传播与网络可靠性。

## 架构图

```
                          +------------------+
                          |   浏览器前端      |
                          | Leaflet + Canvas |
                          +--------+---------+
                                   |
                              HTTP :80
                                   |
                          +--------v---------+
                          |  Nginx 前端容器   |
                          | Gzip / 反代 /api  |
                          +--------+---------+
                                   |
                          +--------v---------+
                          |  Go 后端容器      |
                          | beacon-server    |
                          | Gin :8080        |
                          | pprof :6060      |
                          +--+------+-------+
                             |      |       |
                  +----------+  +---+---+  +----------+
                  |             |       |             |
           +------v------+  +--v---+ +-v--------+ +-v-----------+
           | PostgreSQL  |  | MQTT | |Prometheus | |  模拟器     |
           | + PostGIS   |  |Broker| | (可选)    | | (可选)      |
           | :5432       |  |:1883 | |           | | Python      |
           +-------------+  +------+ +-----------+ +-------------+
```

### 模块架构

```
beacon-server
  ├── dtu_receiver          传感器数据采集和校验
  ├── visibility_analyzer   DEM视线和可视性矩阵计算
  ├── network_reliability   图论和蒙特卡洛模拟
  ├── alarm_mqtt            告警评估和MQTT推送
  └── eventbus              模块间channel通信
       dtu_receiver ──publish──> EventSensorDataReceived
       dtu_receiver ──publish──> EventSignalReceptionReceived
       visibility   ──publish──> EventVisibilityCalculated
       network      ──publish──> EventReliabilityAnalyzed
       network      ──publish──> EventConnectivityCheck
       alarm_mqtt   <──subscribe── 以上全部事件
       alarm_mqtt   ──publish──> EventAlertTriggered
```

## 技术栈

| 层 | 技术 |
|---|---|
| 后端 | Go 1.21+, Gin, sqlx, Prometheus client_golang |
| 数据库 | PostgreSQL 16 + PostGIS 3.4, GIST空间索引 |
| 消息 | Eclipse Mosquitto MQTT 2.0 |
| 前端 | Leaflet 1.9, Canvas, Nginx (Gzip压缩) |
| 可观测 | pprof, Prometheus metrics, /health |
| 模拟器 | Python 3.12, requests |
| 编排 | Docker Compose, 多阶段构建 |

## 部署步骤

### 前提条件

- Docker 20.10+
- Docker Compose v2.0+

### 一键启动

```bash
# 克隆项目
cd beacon-system

# 复制环境变量
cp .env.example .env

# 启动核心服务 (PostgreSQL + MQTT + Go + Nginx)
docker compose up -d

# 查看日志
docker compose logs -f beacon-server

# 等待数据库初始化完成 (约30秒)
# 访问 http://localhost
```

### 启动传感器模拟器

```bash
# 自动天气模式 (默认60秒间隔)
docker compose --profile simulator up -d simulator

# 沙尘暴模式
SIM_WEATHER=sandstorm docker compose --profile simulator up -d simulator

# 固定能见度 2km
SIM_VISIBILITY=2.0 docker compose --profile simulator up -d simulator

# 快速上报 10秒间隔
SIM_INTERVAL=10 docker compose --profile simulator up -d simulator
```

### 本地开发 (无 Docker)

```bash
# 1. 数据库初始化
createdb beacon_system
psql -d beacon_system -f database/init.sql

# 2. 启动 MQTT Broker (可选)
docker run -d -p 1883:1883 eclipse-mosquitto:2.0

# 3. 启动后端
cd backend
export DB_HOST=localhost DB_USER=postgres DB_PASSWORD=postgres DB_NAME=beacon_system
export DEMO_MODE=true
go mod tidy
go run main.go

# 4. 启动前端
cd frontend
python3 -m http.server 3000

# 5. 启动模拟器
cd simulator
pip install requests
python beacon_simulator.py --weather foggy --interval 30
```

## 模拟器用法

传感器模拟器支持多种运行模式，通过命令行参数或环境变量配置：

### 命令行参数

```bash
python beacon_simulator.py [选项]

选项:
  -i, --interval SECONDS    传感器上报间隔，默认60
  -w, --weather MODE        天气模式，默认auto
  -v, --visibility KM       固定能见度(km)，覆盖天气模式
  --api URL                 API地址
  --test                    运行单次测试后退出
```

### 天气模式

| 模式 | 能见度范围 | 风速范围 | 天气因子 |
|------|-----------|---------|---------|
| `auto` | 动态切换 | 动态切换 | 0.2-1.0 |
| `clear` | 15-23 km | 3-7 m/s | 1.0 |
| `light_haze` | 8-12 km | 2-5 m/s | 0.8 |
| `foggy` | 3-6 km | 1-3 m/s | 0.6 |
| `heavy_fog` | 1-2.5 km | 0.5-2 m/s | 0.4 |
| `sandstorm` | 0.3-1 km | 12-18 m/s | 0.2 |

### 使用示例

```bash
# 默认模式: 自动天气，60秒间隔
python beacon_simulator.py

# 沙尘暴场景
python beacon_simulator.py --weather sandstorm

# 浓雾低能见度
python beacon_simulator.py --weather heavy_fog

# 精确控制能见度为 1.5km
python beacon_simulator.py --visibility 1.5

# 大雾模式 + 快速上报10秒
python beacon_simulator.py -w foggy -i 10

# 单次测试
python beacon_simulator.py --test

# Docker 环境变量配置
SIM_WEATHER=sandstorm SIM_INTERVAL=30 docker compose --profile simulator up simulator
```

### Docker 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `API_BASE` | `http://beacon-server:8080/api` | Go后端API地址 |
| `INTERVAL` | `60` | 上报间隔(秒) |
| `WEATHER` | `auto` | 天气模式 |
| `VISIBILITY` | (空) | 固定能见度(km) |

## 可观测性

### pprof 性能分析

```bash
# CPU profile (30秒采样)
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# 内存分配
go tool pprof http://localhost:6060/debug/pprof/heap

# Goroutine 分析
go tool pprof http://localhost:6060/debug/pprof/goroutine

# 30秒 trace
curl -o trace.out http://localhost:6060/debug/pprof/trace?seconds=30
go tool trace trace.out
```

### Prometheus 指标

访问 `http://localhost:6060/metrics` 获取 Prometheus 格式指标：

| 指标 | 类型 | 说明 |
|------|------|------|
| `beacon_sensor_data_received_total` | Counter | 接收的传感器数据总数 |
| `beacon_signal_reception_received_total` | Counter | 接收的信号记录总数 |
| `beacon_visibility_calculations_total` | Counter | 视线计算总数 |
| `beacon_monte_carlo_runs_total` | Counter | 蒙特卡洛模拟总数 |
| `beacon_alerts_triggered_total` | Counter | 触发的告警(按类型/严重度) |
| `beacon_http_requests_total` | Counter | HTTP请求总数(按方法/路径/状态) |
| `beacon_http_duration_seconds` | Histogram | HTTP请求延迟分布 |
| `beacon_monte_carlo_duration_seconds` | Histogram | 蒙特卡洛耗时分布 |
| `beacon_visibility_duration_seconds` | Histogram | 视线计算耗时分布 |
| `beacon_active_beacons` | Gauge | 活跃烽火台数量 |
| `beacon_network_connectivity_index` | Gauge | 当前网络连通度 |
| `beacon_network_reliability` | Gauge | 当前网络可靠性 |
| `beacon_eventbus_published_total` | Counter | 事件总线发布数(按事件类型) |
| `beacon_validation_failures_total` | Counter | 数据校验失败数(按字段) |

### Prometheus 配置示例

```yaml
scrape_configs:
  - job_name: 'beacon-server'
    static_configs:
      - targets: ['localhost:6060']
    metrics_path: '/metrics'
    scrape_interval: 15s
```

## API 接口

### 烽火台管理
- `GET /api/beacons` - 获取所有烽火台
- `GET /api/beacons/:id` - 获取单个烽火台
- `POST /api/beacons` - 创建烽火台

### 传感器数据
- `GET /api/sensor-data` - 获取传感器数据
- `POST /api/sensor-data` - 上报传感器数据
- `GET /api/sensor-data/latest` - 获取最新数据

### 信号接收
- `GET /api/signal-reception` - 获取信号接收记录
- `POST /api/signal-reception` - 上报信号接收状态

### 视线分析
- `GET /api/visibility` - 获取可视性矩阵
- `GET /api/visibility/calculate?from_id=X&to_id=Y` - 计算两点可视性
- `POST /api/visibility/matrix` - 计算完整可视性矩阵
- `GET /api/beacons/:id/viewshed` - 获取视域扇形

### 网络分析
- `GET /api/network/topology` - 获取网络拓扑
- `GET /api/network/reliability?iterations=N&weather_factor=F` - 运行可靠性分析
- `GET /api/network/reliability/history` - 获取分析历史
- `GET /api/network/connectivity` - 检查连通性
- `GET /api/network/critical-links` - 获取关键链路

### 告警管理
- `GET /api/alerts` - 获取告警列表
- `PUT /api/alerts/:id/resolve` - 标记告警已解决

### 系统
- `GET /health` - 健康检查
- `GET /params` - 查看当前参数配置

## 核心算法

### 视线分析 (ITU-R 折射模型)
1. **等效地球半径**: `R_eff = k * R`，k=4/3 (ITU-R P.528)
2. **折射曲率修正**: `drop = d^2 / (2 * R_eff)`
3. **DEM 地形剖面**: 沿视线采样高程，用 params.json 搜索半径
4. **视域扇形**: 方位角步进 + 距离环生成扇区多边形

### 网络可靠性 (重要性采样蒙特卡洛)
1. **图论建模**: 邻接表存储节点和边
2. **蒙特卡洛模拟**: 边按概率随机故障，统计连通概率
3. **重要性采样**: 边数>=20 时自动启用，偏置分布 `1-(1-p)^2`，似然比加权
4. **关键链路**: 迭代移除单边，评估连通度变化量

### 信号波纹 (Canvas 批处理渲染)
1. **进度分桶**: `BATCH_BUCKET_SIZE=0.05`，同桶合并 `beginPath -> N*arc -> stroke`
2. **视口裁剪**: `OFFSCREEN_MARGIN=100px`
3. **队列上限**: `MAX_RIPPLES=200`

## 目录结构

```
beacon-system/
├── backend/                    # Go 后端
│   ├── Dockerfile             # 多阶段构建
│   ├── main.go                # 主入口 (pprof + Prometheus)
│   ├── go.mod
│   ├── config/
│   │   ├── config.go          # 配置 + 参数加载
│   │   └── params.json        # 地形/大气/视域/可靠性/天气参数
│   ├── metrics/
│   │   └── metrics.go         # Prometheus 指标定义
│   ├── models/
│   │   └── models.go
│   ├── database/
│   │   └── db.go
│   ├── handlers/              # HTTP 处理器 (薄层，调用模块)
│   │   ├── beacon.go
│   │   ├── sensor.go
│   │   ├── visibility.go
│   │   └── network.go
│   ├── analysis/              # 核心算法
│   │   ├── visibility.go      # ITU-R 折射视线分析
│   │   └── reliability.go     # 重要性采样蒙特卡洛
│   ├── modules/               # 业务模块
│   │   ├── eventbus/          # Channel 事件总线
│   │   ├── dtu_receiver/      # 传感器采集校验
│   │   ├── visibility_analyzer/  # DEM视线计算
│   │   ├── network_reliability_analyzer/  # 图论+蒙特卡洛
│   │   └── alarm_mqtt/        # 告警评估+MQTT推送
│   └── mqtt/
│       └── client.go
├── frontend/                   # 前端
│   ├── Dockerfile             # Nginx + Gzip
│   ├── nginx.conf             # 反代 + Gzip + 缓存
│   ├── index.html
│   ├── css/style.css
│   └── js/
│       ├── beacon_tower_map.js   # 地图+烽火台+拓扑
│       ├── visibility_panel.js   # 视域分析面板
│       ├── signal.js             # 信号波纹(批处理)
│       └── app.js                # 主应用+可靠性分析
├── database/
│   ├── Dockerfile             # PostGIS 16-3.4
│   └── init.sql               # 建表+空间索引+样例数据+CLUSTER+VACUUM
├── simulator/
│   ├── Dockerfile             # Python 3.12-slim
│   └── beacon_simulator.py    # 增强模拟器(argparse+天气+能见度)
├── mqtt/
│   └── config/
│       └── mosquitto.conf     # MQTT Broker 配置
├── docker-compose.yml          # 全服务编排
├── .env                        # 环境变量
└── .env.example                # 环境变量示例
```

## 配置说明

### 环境变量

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `SERVER_PORT` | 8080 | Go服务端口 |
| `DB_HOST` | postgres | 数据库地址 |
| `DB_PORT` | 5432 | 数据库端口 |
| `DB_USER` | postgres | 数据库用户 |
| `DB_PASSWORD` | postgres | 数据库密码 |
| `DB_NAME` | beacon_system | 数据库名 |
| `MQTT_BROKER` | mqtt-broker | MQTT代理地址 |
| `MQTT_PORT` | 1883 | MQTT端口 |
| `MQTT_TOPIC` | beacon/alerts | 告警主题 |
| `CONNECTIVITY_THRESHOLD` | 0.7 | 连通度告警阈值 |
| `DEMO_MODE` | false | 演示模式(禁用MQTT) |
| `SIM_INTERVAL` | 60 | 模拟器上报间隔 |
| `SIM_WEATHER` | auto | 模拟器天气模式 |
| `SIM_VISIBILITY` | (空) | 模拟器固定能见度 |

### 参数配置文件 (config/params.json)

| 类别 | 参数 | 默认值 | 说明 |
|------|------|--------|------|
| 地形 | `dem_resolution_meters` | 30 | DEM分辨率 |
| 地形 | `dem_search_radius_meters` | 5000 | DEM搜索半径 |
| 地形 | `max_analysis_distance_km` | 100 | 最大分析距离 |
| 大气 | `itu_r_effective_earth_factor_k` | 1.333 | ITU-R等效地球因子 |
| 大气 | `refraction_gradient_n_units_per_km` | -40 | 折射梯度 |
| 大气 | `default_refraction_model` | standard | 默认折射模型 |
| 视域 | `default_viewshed_distance_km` | 20 | 默认视域距离 |
| 视域 | `viewshed_azimuth_steps` | 36 | 方位角步数 |
| 可靠性 | `default_monte_carlo_iterations` | 1000 | 默认MC迭代次数 |
| 可靠性 | `importance_sampling_edge_threshold` | 20 | IS启用边阈值 |
| 可靠性 | `importance_sampling_bias_factor` | 2.0 | IS偏置因子 |
| 可靠性 | `connectivity_warning_threshold` | 0.7 | 连通度告警阈值 |

## 许可证

本项目仅供学术研究和教育用途。
