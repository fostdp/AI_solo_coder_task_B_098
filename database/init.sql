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

-- ============================================================
-- 13. 空间索引优化和统计信息
-- ============================================================

ANALYZE beacons;
ANALYZE dem_data;
ANALYZE view_sheds;

CLUSTER beacons USING idx_beacons_location;
CLUSTER dem_data USING idx_dem_data_geom;
CLUSTER view_sheds USING idx_viewsheds_geom;

ALTER TABLE dem_data CLUSTER ON idx_dem_data_geom;
ALTER TABLE view_sheds CLUSTER ON idx_viewsheds_geom;

SET work_mem = '256MB';
SET maintenance_work_mem = '512MB';

VACUUM ANALYZE beacons;
VACUUM ANALYZE dem_data;
VACUUM ANALYZE sensor_data;
VACUUM ANALYZE signal_reception;
VACUUM ANALYZE visibility_analysis;
VACUUM ANALYZE network_links;
VACUUM ANALYZE reliability_analysis;
VACUUM ANALYZE alerts;
VACUUM ANALYZE view_sheds;

-- ============================================================
-- 14. 朝代表：不同朝代的烽火台体系
-- ============================================================
CREATE TABLE IF NOT EXISTS dynasties (
    code VARCHAR(20) PRIMARY KEY,
    name VARCHAR(50) NOT NULL,
    period VARCHAR(100),
    description TEXT,
    color VARCHAR(20) DEFAULT '#e94560',
    sort_order INTEGER DEFAULT 0
);

INSERT INTO dynasties (code, name, period, description, color, sort_order) VALUES
('qin',  '秦代', '公元前221-前207年', '秦代长城烽火台，初创体系，以渭河流域为核心', '#8b5cf6', 1),
('han',  '汉代', '公元前202-公元220年', '汉代河西走廊烽火台，丝绸之路保障', '#ef4444', 2),
('ming', '明代', '公元1368-1644年', '明代九边重镇烽火台，防御蒙古', '#f59e0b', 3)
ON CONFLICT (code) DO NOTHING;

-- 为 network_topology 增加朝代字段（兼容汉代默认值）
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='network_topology' AND column_name='dynasty_code') THEN
        ALTER TABLE network_topology ADD COLUMN dynasty_code VARCHAR(20) DEFAULT 'han' REFERENCES dynasties(code);
    END IF;
END $$;

-- 为 network_topology 增加名称字段
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='network_topology' AND column_name='name') THEN
        ALTER TABLE network_topology ADD COLUMN name VARCHAR(100);
    END IF;
END $$;

-- 更新现有拓扑
UPDATE network_topology SET name = '汉代河西走廊主线烽火台网络', dynasty_code = 'han' WHERE id = 1;

-- ============================================================
-- 15. 现代基站表：跨时代对比用
-- ============================================================
CREATE TABLE IF NOT EXISTS modern_base_stations (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    station_type VARCHAR(20) DEFAULT '5g',
    location GEOGRAPHY(POINT, 4326) NOT NULL,
    height NUMERIC(6, 2) DEFAULT 30.0,
    coverage_radius_km NUMERIC(6, 2) DEFAULT 1.5,
    capacity_mbps NUMERIC(8, 2) DEFAULT 1000.0,
    latency_ms NUMERIC(6, 2) DEFAULT 10.0,
    frequency_ghz NUMERIC(5, 2) DEFAULT 3.5,
    power_kw NUMERIC(5, 2) DEFAULT 1.2,
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_modern_stations_location ON modern_base_stations USING GIST(location);
CREATE INDEX IF NOT EXISTS idx_modern_stations_type ON modern_base_stations(station_type);

-- ============================================================
-- 16. 抗毁性分析结果表
-- ============================================================
CREATE TABLE IF NOT EXISTS resilience_analysis (
    id SERIAL PRIMARY KEY,
    topology_id INTEGER REFERENCES network_topology(id),
    analysis_type VARCHAR(30) NOT NULL,
    attack_type VARCHAR(30) NOT NULL,
    node_removal_ratio NUMERIC(5, 4) NOT NULL,
    connectivity_index NUMERIC(6, 4),
    giant_component_size INTEGER,
    robustness_score NUMERIC(6, 4),
    iterations INTEGER DEFAULT 1,
    details JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_resilience_topology ON resilience_analysis(topology_id);
CREATE INDEX IF NOT EXISTS idx_resilience_created ON resilience_analysis(created_at DESC);

-- ============================================================
-- 17. 用户虚拟点燃记录表
-- ============================================================
CREATE TABLE IF NOT EXISTS user_beacon_ignitions (
    id BIGSERIAL PRIMARY KEY,
    session_id VARCHAR(64),
    beacon_id INTEGER REFERENCES beacons(id),
    topology_id INTEGER REFERENCES network_topology(id),
    user_note VARCHAR(200),
    weather_factor NUMERIC(5, 4) DEFAULT 1.0,
    reached_count INTEGER DEFAULT 0,
    total_propagation_time_ms NUMERIC(12, 2),
    propagation_path JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_ignitions_beacon ON user_beacon_ignitions(beacon_id);
CREATE INDEX IF NOT EXISTS idx_ignitions_session ON user_beacon_ignitions(session_id);
CREATE INDEX IF NOT EXISTS idx_ignitions_created ON user_beacon_ignitions(created_at DESC);

-- ============================================================
-- 18. 朝代样例数据：秦代烽火台（关中-渭河沿线）
-- ============================================================
INSERT INTO beacons (name, code, dynasty, location, elevation, height, description, status)
VALUES
('咸阳烽火台', 'XY-QIN-01', '秦代', ST_SetSRID(ST_MakePoint(108.71, 34.33), 4326), 400.0, 8.0, '秦都咸阳近郊烽火台', 'active'),
('临潼烽火台', 'LT-QIN-02', '秦代', ST_SetSRID(ST_MakePoint(109.22, 34.37), 4326), 450.0, 9.0, '临潼新丰镇烽火台', 'active'),
('华阴烽火台', 'HY-QIN-03', '秦代', ST_SetSRID(ST_MakePoint(110.09, 34.56), 4326), 520.0, 8.5, '华阴潼关前线烽火台', 'active'),
('宝鸡烽火台', 'BJ-QIN-04', '秦代', ST_SetSRID(ST_MakePoint(107.15, 34.37), 4326), 620.0, 7.5, '宝鸡陈仓烽火台', 'active'),
('天水烽火台', 'TS-QIN-05', '秦代', ST_SetSRID(ST_MakePoint(105.72, 34.58), 4326), 1140.0, 10.0, '天水陇西烽火台', 'active'),
('延安烽火台', 'YA-QIN-06', '秦代', ST_SetSRID(ST_MakePoint(109.49, 36.59), 4326), 1100.0, 9.0, '延安北境烽火台', 'active'),
('韩城烽火台', 'HC-QIN-07', '秦代', ST_SetSRID(ST_MakePoint(110.45, 35.48), 4326), 550.0, 8.0, '韩城黄河边烽火台', 'active'),
('汉中烽火台', 'HZ-QIN-08', '秦代', ST_SetSRID(ST_MakePoint(107.03, 33.07), 4326), 550.0, 9.5, '汉中蜀道烽火台', 'active')
ON CONFLICT (code) DO NOTHING;

-- 秦代网络拓扑
INSERT INTO network_topology (version, name, description, is_active, dynasty_code)
VALUES (1, '秦代关中烽火台网络', '秦代渭河沿线及北境防御体系', false, 'qin')
ON CONFLICT DO NOTHING;

INSERT INTO network_links (topology_id, from_beacon_id, to_beacon_id, link_type, base_reliability, is_bidirectional, is_critical)
VALUES
(2, 13, 14, 'visual', 0.90, true, true),
(2, 14, 15, 'visual', 0.92, true, false),
(2, 13, 16, 'visual', 0.85, true, false),
(2, 16, 17, 'visual', 0.88, true, false),
(2, 14, 18, 'visual', 0.87, true, false),
(2, 15, 19, 'visual', 0.91, true, false),
(2, 13, 20, 'visual', 0.83, true, false)
ON CONFLICT DO NOTHING;

-- ============================================================
-- 19. 朝代样例数据：明代九边烽火台（更密集）
-- ============================================================
INSERT INTO beacons (name, code, dynasty, location, elevation, height, description, status)
VALUES
('嘉峪关明台', 'JYG-M-01', '明代', ST_SetSRID(ST_MakePoint(98.29, 39.77), 4326), 1666.0, 14.0, '嘉峪关关城明代烽火台', 'active'),
('酒泉明台', 'JQ-M-02', '明代', ST_SetSRID(ST_MakePoint(98.50, 39.73), 4326), 1480.0, 11.0, '酒泉明代卫所烽火台', 'active'),
('张掖明台', 'ZY-M-03', '明代', ST_SetSRID(ST_MakePoint(100.46, 38.93), 4326), 1485.0, 12.0, '张掖甘州明代烽火台', 'active'),
('山丹明台', 'SD-M-04', '明代', ST_SetSRID(ST_MakePoint(101.09, 38.79), 4326), 1760.0, 10.5, '山丹峡口明代烽火台', 'active'),
('武威明台', 'WW-M-05', '明代', ST_SetSRID(ST_MakePoint(102.64, 37.93), 4326), 1530.0, 13.0, '武威凉州明代烽火台', 'active'),
('古浪明台', 'GL-M-06', '明代', ST_SetSRID(ST_MakePoint(102.90, 37.48), 4326), 2000.0, 10.0, '古浪乌鞘岭明代烽火台', 'active'),
('兰州明台', 'LZ-M-07', '明代', ST_SetSRID(ST_MakePoint(103.83, 36.06), 4326), 1520.0, 12.5, '兰州金城关明代烽火台', 'active'),
('临洮明台', 'LT-M-08', '明代', ST_SetSRID(ST_MakePoint(103.86, 35.37), 4326), 1880.0, 9.5, '临洮明代边墙烽火台', 'active'),
('固原明台', 'GY-M-09', '明代', ST_SetSRID(ST_MakePoint(106.28, 36.02), 4326), 1800.0, 11.0, '固原明代总镇烽火台', 'active'),
('银川明台', 'YC-M-10', '明代', ST_SetSRID(ST_MakePoint(106.27, 38.47), 4326), 1110.0, 12.0, '银川宁夏镇明代烽火台', 'active')
ON CONFLICT (code) DO NOTHING;

-- 明代网络拓扑
INSERT INTO network_topology (version, name, description, is_active, dynasty_code)
VALUES (1, '明代九边河西段烽火台网络', '明代九边重镇甘肃镇防御体系，更密集更规范', false, 'ming')
ON CONFLICT DO NOTHING;

INSERT INTO network_links (topology_id, from_beacon_id, to_beacon_id, link_type, base_reliability, is_bidirectional, is_critical)
VALUES
(3, 21, 22, 'visual', 0.94, true, true),
(3, 22, 23, 'visual', 0.91, true, false),
(3, 23, 24, 'visual', 0.93, true, false),
(3, 24, 25, 'visual', 0.90, true, false),
(3, 25, 26, 'visual', 0.92, true, false),
(3, 26, 27, 'visual', 0.89, true, true),
(3, 27, 28, 'visual', 0.91, true, false),
(3, 27, 29, 'visual', 0.87, true, false),
(3, 29, 30, 'visual', 0.85, true, false),
(3, 23, 26, 'visual', 0.82, true, false)
ON CONFLICT DO NOTHING;

-- ============================================================
-- 20. 现代基站样例数据（河西走廊沿线5G基站模拟）
-- ============================================================
INSERT INTO modern_base_stations (name, station_type, location, height, coverage_radius_km, capacity_mbps, latency_ms, frequency_ghz, power_kw, status)
VALUES
('玉门5G基站', '5g', ST_SetSRID(ST_MakePoint(97.04, 40.29), 4326), 35.0, 1.2, 1200.0, 8.0, 3.5, 1.5, 'active'),
('嘉峪关5G基站', '5g', ST_SetSRID(ST_MakePoint(98.29, 39.77), 4326), 40.0, 1.5, 1500.0, 7.0, 3.5, 2.0, 'active'),
('酒泉5G基站', '5g', ST_SetSRID(ST_MakePoint(98.50, 39.73), 4326), 32.0, 1.0, 1000.0, 9.0, 2.6, 1.2, 'active'),
('张掖5G基站', '5g', ST_SetSRID(ST_MakePoint(100.46, 38.93), 4326), 38.0, 1.8, 1800.0, 6.5, 3.5, 2.5, 'active'),
('武威5G基站', '5g', ST_SetSRID(ST_MakePoint(102.64, 37.93), 4326), 30.0, 1.3, 1300.0, 8.5, 2.6, 1.8, 'active'),
('兰州5G基站', '5g', ST_SetSRID(ST_MakePoint(103.83, 36.06), 4326), 45.0, 2.0, 2000.0, 5.0, 3.5, 3.0, 'active'),
('天水5G基站', '5g', ST_SetSRID(ST_MakePoint(105.72, 34.58), 4326), 35.0, 1.5, 1400.0, 7.5, 3.5, 2.0, 'active'),
('西安5G基站', '5g', ST_SetSRID(ST_MakePoint(108.94, 34.27), 4326), 50.0, 2.5, 3000.0, 4.0, 3.5, 5.0, 'active'),
('敦煌微波站', 'microwave', ST_SetSRID(ST_MakePoint(94.66, 40.14), 4326), 80.0, 50.0, 500.0, 2.0, 6.0, 0.5, 'active'),
('西宁卫星站', 'satellite', ST_SetSRID(ST_MakePoint(101.78, 36.62), 4326), 200.0, 200.0, 100.0, 500.0, 14.0, 10.0, 'active')
ON CONFLICT DO NOTHING;

-- ============================================================
-- 完成
-- ============================================================
ANALYZE dynasties;
ANALYZE modern_base_stations;
ANALYZE resilience_analysis;
ANALYZE user_beacon_ignitions;

VACUUM ANALYZE dynasties;
VACUUM ANALYZE modern_base_stations;
VACUUM ANALYZE resilience_analysis;
VACUUM ANALYZE user_beacon_ignitions;
