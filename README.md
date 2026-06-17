# 古代烽火台视线分析与通信网络可靠性仿真系统

## 项目简介

本系统是一套用于研究汉代河西走廊烽火台体系的全栈仿真应用。通过数字化建模和仿真技术，还原古代军事通信网络的运作机制，分析视线传播与网络可靠性。

## 技术栈

### 后端
- **语言**: Go 1.21+
- **Web框架**: Gin
- **数据库**: PostgreSQL + PostGIS
- **ORM**: sqlx
- **消息队列**: MQTT (Eclipse Paho)

### 前端
- **地图**: Leaflet
- **绘图**: HTML5 Canvas
- **样式**: 原生 CSS

### 数据模拟
- **传感器模拟器**: Python 3.x
- **数据上报**: REST API

## 系统功能

### 1. 视线分析模型
- 基于 DEM 数字高程数据计算可视性
- 考虑地球曲率影响
- 计算可视距离、方位角、仰角等参数
- 生成视域扇形区域

### 2. 通信网络可靠性分析
- 基于图论的网络拓扑建模
- 蒙特卡洛模拟评估信号传递成功率
- 关键链路识别
- 天气干扰下的可靠性评估

### 3. 告警系统
- 关键链路中断告警
- 网络连通度低于阈值预警
- MQTT 消息推送
- 告警记录管理

### 4. 可视化展示
- 烽火台分布地图 (Leaflet)
- 视域半透明扇形标注
- 信号传播动态波纹动画
- 实时数据面板

### 5. 传感器模拟
- 每分钟模拟传感器数据上报
- 能见度、风速、温度、湿度模拟
- 天气模式动态变化
- 相邻烽火台信号接收状态模拟

## 目录结构

```
beacon-system/
├── backend/                 # Go 后端
│   ├── main.go             # 主程序入口
│   ├── go.mod              # 依赖管理
│   ├── config/             # 配置模块
│   ├── models/             # 数据模型
│   ├── database/           # 数据库连接
│   ├── handlers/           # HTTP 处理器
│   ├── analysis/           # 分析算法
│   │   ├── visibility.go   # 视线分析
│   │   └── reliability.go  # 网络可靠性分析
│   └── mqtt/               # MQTT 客户端
├── frontend/                # 前端
│   ├── index.html          # 主页面
│   ├── css/style.css       # 样式
│   └── js/                 # JavaScript
│       ├── map.js          # 地图核心
│       ├── visibility.js   # 视域分析
│       ├── signal.js       # 信号动画
│       └── app.js          # 主应用
├── database/                # 数据库
│   └── init.sql            # 初始化脚本
└── simulator/               # 模拟器
    └── beacon_simulator.py # 传感器模拟器
```

## 快速开始

### 1. 数据库初始化

```bash
# 创建数据库
createdb beacon_system

# 执行初始化脚本
psql -d beacon_system -f database/init.sql
```

### 2. 后端启动

```bash
cd backend

# 设置环境变量 (或复制 .env.example 为 .env)
export DB_HOST=localhost
export DB_USER=postgres
export DB_PASSWORD=your_password
export DB_NAME=beacon_system
export DEMO_MODE=true

# 安装依赖
go mod tidy

# 运行
go run main.go
```

后端服务将在 `http://localhost:8080` 启动

### 3. 前端访问

直接在浏览器打开 `frontend/index.html` 文件即可。

或使用简单的 HTTP 服务器:

```bash
cd frontend
python3 -m http.server 3000
# 访问 http://localhost:3000
```

### 4. 传感器模拟器

```bash
cd simulator

# 安装依赖
pip install requests

# 运行 (默认60秒上报一次)
python beacon_simulator.py

# 自定义间隔 (30秒)
python beacon_simulator.py 30

# 单次测试
python beacon_simulator.py --test
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
- `GET /api/visibility/calculate` - 计算两点可视性
- `POST /api/visibility/matrix` - 计算完整可视性矩阵
- `GET /api/beacons/:id/viewshed` - 获取视域扇形

### 网络分析
- `GET /api/network/topology` - 获取网络拓扑
- `GET /api/network/reliability` - 运行可靠性分析
- `GET /api/network/reliability/history` - 获取分析历史
- `GET /api/network/connectivity` - 检查连通性
- `GET /api/network/critical-links` - 获取关键链路

### 告警管理
- `GET /api/alerts` - 获取告警列表
- `PUT /api/alerts/:id/resolve` - 标记告警已解决

## 核心算法

### 视线分析算法
1. **Haversine 距离公式**: 计算两点间地表距离
2. **地球曲率修正**: `drop = d² / (2R)`
3. **DEM 插值**: 沿视线采样地形高程
4. **视线高度计算**: 考虑地球曲率的视线高度剖面

### 网络可靠性分析
1. **图论建模**: 烽火台为节点，视线链路为边
2. **BFS 广度优先搜索**: 计算连通性和最短路径
3. **蒙特卡洛模拟**: 随机模拟链路故障，统计连通概率
4. **关键链路识别**: 移除单条链路后网络连通性变化评估

### 天气影响模型
- 能见度因子: 与可视距离正相关
- 风速因子: 风速越高信号衰减越大
- 综合天气因子: 影响链路可靠性参数

## 配置说明

### 环境变量

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| SERVER_PORT | 8080 | 服务端口 |
| DB_HOST | localhost | 数据库地址 |
| DB_PORT | 5432 | 数据库端口 |
| DB_USER | postgres | 数据库用户 |
| DB_PASSWORD | postgres | 数据库密码 |
| DB_NAME | beacon_system | 数据库名 |
| MQTT_BROKER | localhost | MQTT 代理地址 |
| MQTT_PORT | 1883 | MQTT 端口 |
| MQTT_TOPIC | beacon/alerts | 告警主题 |
| CONNECTIVITY_THRESHOLD | 0.7 | 连通度告警阈值 |
| DEMO_MODE | true | 演示模式 (禁用MQTT) |

## 数据库表结构

- **beacons**: 烽火台基础信息
- **dem_data**: DEM 高程数据
- **sensor_data**: 传感器实时数据
- **signal_reception**: 信号接收状态
- **visibility_analysis**: 视线分析结果
- **network_topology**: 网络拓扑
- **network_links**: 网络链路
- **reliability_analysis**: 可靠性分析结果
- **alerts**: 告警记录
- **view_sheds**: 视域区域

## 许可证

本项目仅供学术研究和教育用途。
