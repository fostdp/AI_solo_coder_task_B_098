-- ============================================================
-- 古代烽火台视线分析与通信网络可靠性仿真系统
-- PostgreSQL + PostGIS 数据库初始化脚本
-- ============================================================

-- 启用 PostGIS 扩展
CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS postgis_topology;

-- ============================================================
-- 1. 烽火台基础信息表
-- ============================================================
CREATE TABLE IF NOT EXISTS beacons (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    code VARCHAR(50) UNIQUE NOT NULL,
    dynasty VARCHAR(50) DEFAULT '汉代',
    location GEOGRAPHY(POINT, 4326) NOT NULL,
    elevation NUMERIC(10, 2) NOT NULL,
    height NUMERIC(6, 2) DEFAULT 10.0,
    description TEXT,
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_beacons_location ON beacons USING GIST(location);
CREATE INDEX IF NOT EXISTS idx_beacons_status ON beacons(status);

-- ============================================================
-- 2. 数字高程模型(DEM)数据表
-- ============================================================
CREATE TABLE IF NOT EXISTS dem_data (
    id SERIAL PRIMARY KEY,
    lon NUMERIC(10, 6) NOT NULL,
    lat NUMERIC(10, 6) NOT NULL,
    elevation NUMERIC(10, 2) NOT NULL,
    resolution NUMERIC(8, 2) DEFAULT 30.0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_dem_data_geom ON dem_data USING GIST(ST_SetSRID(ST_MakePoint(lon, lat), 4326));
CREATE INDEX IF NOT EXISTS idx_dem_data_lon_lat ON dem_data(lon, lat);

-- ============================================================
-- 3. 传感器实时数据表
-- ============================================================
CREATE TABLE IF NOT EXISTS sensor_data (
    id BIGSERIAL PRIMARY KEY,
    beacon_id INTEGER NOT NULL REFERENCES beacons(id) ON DELETE CASCADE,
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    visibility NUMERIC(8, 2) NOT NULL,
    wind_speed NUMERIC(6, 2) NOT NULL,
    wind_direction NUMERIC(5, 2),
    temperature NUMERIC(5, 2),
    humidity NUMERIC(5, 2),
    terrain_elevation NUMERIC(10, 2)
);

CREATE INDEX IF NOT EXISTS idx_sensor_data_beacon_time ON sensor_data(beacon_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_sensor_data_timestamp ON sensor_data(timestamp DESC);

-- ============================================================
-- 4. 相邻烽火台信号接收状态表
-- ============================================================
CREATE TABLE IF NOT EXISTS signal_reception (
    id BIGSERIAL PRIMARY KEY,
    from_beacon_id INTEGER NOT NULL REFERENCES beacons(id) ON DELETE CASCADE,
    to_beacon_id INTEGER NOT NULL REFERENCES beacons(id) ON DELETE CASCADE,
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    signal_strength NUMERIC(5, 2) NOT NULL,
    is_received BOOLEAN DEFAULT false,
    interference_level NUMERIC(5, 2) DEFAULT 0,
    weather_factor NUMERIC(5, 4) DEFAULT 1.0
);

CREATE INDEX IF NOT EXISTS idx_signal_reception_from_to ON signal_reception(from_beacon_id, to_beacon_id);
CREATE INDEX IF NOT EXISTS idx_signal_reception_timestamp ON signal_reception(timestamp DESC);

-- ============================================================
-- 5. 视线分析结果表
-- ============================================================
CREATE TABLE IF NOT EXISTS visibility_analysis (
    id SERIAL PRIMARY KEY,
    from_beacon_id INTEGER NOT NULL REFERENCES beacons(id) ON DELETE CASCADE,
    to_beacon_id INTEGER NOT NULL REFERENCES beacons(id) ON DELETE CASCADE,
    is_visible BOOLEAN NOT NULL DEFAULT false,
    distance_km NUMERIC(10, 3) NOT NULL,
    bearing NUMERIC(7, 3),
    earth_curvature_drop NUMERIC(8, 2),
    min_clearance NUMERIC(8, 2),
    max_terrain_elevation NUMERIC(10, 2),
    visibility_angle NUMERIC(8, 4),
    calculation_method VARCHAR(50) DEFAULT 'dem_line_of_sight',
    calculated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(from_beacon_id, to_beacon_id)
);

CREATE INDEX IF NOT EXISTS idx_visibility_from ON visibility_analysis(from_beacon_id);
CREATE INDEX IF NOT EXISTS idx_visibility_to ON visibility_analysis(to_beacon_id);
CREATE INDEX IF NOT EXISTS idx_visibility_visible ON visibility_analysis(is_visible);

-- ============================================================
-- 6. 通信网络拓扑表
-- ============================================================
CREATE TABLE IF NOT EXISTS network_topology (
    id SERIAL PRIMARY KEY,
    version INTEGER DEFAULT 1,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT false
);

CREATE TABLE IF NOT EXISTS network_links (
    id SERIAL PRIMARY KEY,
    topology_id INTEGER NOT NULL REFERENCES network_topology(id) ON DELETE CASCADE,
    from_beacon_id INTEGER NOT NULL REFERENCES beacons(id) ON DELETE CASCADE,
    to_beacon_id INTEGER NOT NULL REFERENCES beacons(id) ON DELETE CASCADE,
    link_type VARCHAR(20) DEFAULT 'visual',
    capacity NUMERIC(8, 2) DEFAULT 1.0,
    base_reliability NUMERIC(5, 4) DEFAULT 0.95,
    is_bidirectional BOOLEAN DEFAULT true,
    is_critical BOOLEAN DEFAULT false,
    UNIQUE(topology_id, from_beacon_id, to_beacon_id)
);

CREATE INDEX IF NOT EXISTS idx_network_links_topology ON network_links(topology_id);

-- ============================================================
-- 7. 网络可靠性分析结果表
-- ============================================================
CREATE TABLE IF NOT EXISTS reliability_analysis (
    id SERIAL PRIMARY KEY,
    topology_id INTEGER REFERENCES network_topology(id),
    analysis_type VARCHAR(50) NOT NULL,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    overall_reliability NUMERIC(6, 4),
    connectivity_index NUMERIC(6, 4),
    average_path_length NUMERIC(8, 4),
    node_count INTEGER,
    link_count INTEGER,
    monte_carlo_iterations INTEGER,
    weather_condition VARCHAR(50) DEFAULT 'clear',
    details JSONB
);

CREATE INDEX IF NOT EXISTS idx_reliability_timestamp ON reliability_analysis(timestamp DESC);

-- ============================================================
-- 8. 告警记录表
-- ============================================================
CREATE TABLE IF NOT EXISTS alerts (
    id BIGSERIAL PRIMARY KEY,
    alert_type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) NOT NULL,
    title VARCHAR(200) NOT NULL,
    description TEXT,
    beacon_id INTEGER REFERENCES beacons(id),
    link_id INTEGER REFERENCES network_links(id),
    related_data JSONB,
    is_resolved BOOLEAN DEFAULT false,
    resolved_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_alerts_severity ON alerts(severity);
CREATE INDEX IF NOT EXISTS idx_alerts_status ON alerts(is_resolved);
CREATE INDEX IF NOT EXISTS idx_alerts_created ON alerts(created_at DESC);

-- ============================================================
-- 9. 视域扇形区域表（缓存计算结果）
-- ============================================================
CREATE TABLE IF NOT EXISTS view_sheds (
    id SERIAL PRIMARY KEY,
    beacon_id INTEGER NOT NULL REFERENCES beacons(id) ON DELETE CASCADE,
    azimuth_start NUMERIC(6, 2) NOT NULL,
    azimuth_end NUMERIC(6, 2) NOT NULL,
    max_distance NUMERIC(8, 2) NOT NULL,
    geometry GEOGRAPHY(POLYGON, 4326),
    calculated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(beacon_id, azimuth_start, azimuth_end, max_distance)
);

CREATE INDEX IF NOT EXISTS idx_viewsheds_beacon ON view_sheds(beacon_id);
CREATE INDEX IF NOT EXISTS idx_viewsheds_geom ON view_sheds USING GIST(geometry);

-- ============================================================
-- 10. 自动更新时间戳的触发器函数
-- ============================================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS update_beacons_updated_at ON beacons;
CREATE TRIGGER update_beacons_updated_at
    BEFORE UPDATE ON beacons
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================
-- 11. 网络连通度阈值检测函数
-- ============================================================
CREATE OR REPLACE FUNCTION check_network_connectivity(topology_id INTEGER)
RETURNS NUMERIC AS $$
DECLARE
    connectivity NUMERIC;
    node_count INTEGER;
    reachable_count INTEGER;
BEGIN
    SELECT COUNT(DISTINCT beacon_id) INTO node_count
    FROM beacons WHERE status = 'active';

    WITH RECURSIVE reachable_nodes AS (
        SELECT from_beacon_id as node_id
        FROM network_links
        WHERE topology_id = $1
        AND is_bidirectional = true
        LIMIT 1
        UNION
        SELECT
            CASE
                WHEN rn.node_id = nl.from_beacon_id THEN nl.to_beacon_id
                ELSE nl.from_beacon_id
            END as node_id
        FROM reachable_nodes rn
        JOIN network_links nl ON
            (rn.node_id = nl.from_beacon_id OR rn.node_id = nl.to_beacon_id)
        WHERE nl.topology_id = $1
    )
    SELECT COUNT(DISTINCT node_id) INTO reachable_count FROM reachable_nodes;

    IF node_count > 0 THEN
        connectivity := reachable_count::NUMERIC / node_count::NUMERIC;
    ELSE
        connectivity := 0;
    END IF;

    RETURN connectivity;
END;
$$ LANGUAGE plpgsql;

-- ============================================================
-- 12. 初始样例数据：汉代河西走廊烽火台
-- ============================================================
INSERT INTO beacons (name, code, dynasty, location, elevation, height, description, status)
VALUES
('玉门关烽火台', 'YMG-001', '汉代', ST_SetSRID(ST_MakePoint(93.88, 40.35), 4326), 1250.0, 12.0, '玉门关遗址主烽火台，河西走廊西端起点', 'active'),
('河仓城烽火台', 'HCC-002', '汉代', ST_SetSRID(ST_MakePoint(94.02, 40.42), 4326), 1235.0, 10.0, '河仓城附近烽火台', 'active'),
('大方盘城烽火台', 'DFP-003', '汉代', ST_SetSRID(ST_MakePoint(94.15, 40.48), 4326), 1220.0, 11.0, '大方盘城遗址烽火台', 'active'),
('敦煌市烽火台', 'DHS-004', '汉代', ST_SetSRID(ST_MakePoint(94.66, 40.14), 4326), 1139.0, 10.0, '敦煌郡治所附近烽火台', 'active'),
('莫高窟烽火台', 'MGK-005', '汉代', ST_SetSRID(ST_MakePoint(94.81, 40.04), 4326), 1150.0, 9.5, '莫高窟附近警戒烽火台', 'active'),
('瓜州烽火台', 'GZ-006', '汉代', ST_SetSRID(ST_MakePoint(95.78, 40.54), 4326), 1178.0, 10.5, '瓜州（安西）烽火台', 'active'),
('嘉峪关烽火台', 'JYG-007', '汉代', ST_SetSRID(ST_MakePoint(98.29, 39.77), 4326), 1666.0, 12.0, '嘉峪关关城烽火台', 'active'),
('酒泉烽火台', 'JQ-008', '汉代', ST_SetSRID(ST_MakePoint(98.50, 39.73), 4326), 1480.0, 10.0, '酒泉郡烽火台', 'active'),
('张掖烽火台', 'ZY-009', '汉代', ST_SetSRID(ST_MakePoint(100.46, 38.93), 4326), 1485.0, 11.0, '张掖郡治觻得烽火台', 'active'),
('武威烽火台', 'WW-010', '汉代', ST_SetSRID(ST_MakePoint(102.64, 37.93), 4326), 1530.0, 10.0, '武威郡姑臧烽火台', 'active'),
('兰州烽火台', 'LZ-011', '汉代', ST_SetSRID(ST_MakePoint(103.83, 36.06), 4326), 1520.0, 11.5, '金城郡治所烽火台', 'active'),
('天水烽火台', 'TS-012', '汉代', ST_SetSRID(ST_MakePoint(105.72, 34.58), 4326), 1140.0, 9.0, '天水郡烽火台', 'active');

-- 创建初始网络拓扑
INSERT INTO network_topology (version, description, is_active)
VALUES (1, '汉代河西走廊主线烽火台网络', true);

-- 插入相邻链路（假设相邻烽火台之间有直接视线联系）
INSERT INTO network_links (topology_id, from_beacon_id, to_beacon_id, link_type, base_reliability, is_critical)
VALUES
(1, 1, 2, 'visual', 0.92, true),
(1, 2, 3, 'visual', 0.93, false),
(1, 3, 4, 'visual', 0.88, false),
(1, 4, 5, 'visual', 0.91, false),
(1, 5, 6, 'visual', 0.89, false),
(1, 6, 7, 'visual', 0.85, true),
(1, 7, 8, 'visual', 0.94, true),
(1, 8, 9, 'visual', 0.90, false),
(1, 9, 10, 'visual', 0.87, false),
(1, 10, 11, 'visual', 0.86, false),
(1, 11, 12, 'visual', 0.88, false);

-- 初始化一些DEM样例数据（河西走廊区域）
INSERT INTO dem_data (lon, lat, elevation, resolution)
SELECT
    93.5 + (x * 0.5) as lon,
    34.0 + (y * 0.5) as lat,
    1000 + 600 * sin(x * 0.8) * cos(y * 0.6) + random() * 200 as elevation,
    1000.0 as resolution
FROM generate_series(0, 25) as x,
     generate_series(0, 15) as y;

-- 初始化一些传感器历史数据（最近24小时）
INSERT INTO sensor_data (beacon_id, timestamp, visibility, wind_speed, wind_direction, temperature, humidity, terrain_elevation)
SELECT
    b.id,
    NOW() - (random() * interval '24 hours'),
    8.0 + random() * 12.0,
    2.0 + random() * 8.0,
    random() * 360,
    10.0 + random() * 15.0,
    30.0 + random() * 40.0,
    b.elevation
FROM beacons b, generate_series(1, 50);

-- ============================================================
-- 完成
-- ============================================================
COMMENT ON TABLE beacons IS '烽火台基础信息表';
COMMENT ON TABLE sensor_data IS '传感器实时数据表';
COMMENT ON TABLE visibility_analysis IS '视线分析结果表';
COMMENT ON TABLE reliability_analysis IS '网络可靠性分析结果表';
COMMENT ON TABLE alerts IS '告警记录表';
