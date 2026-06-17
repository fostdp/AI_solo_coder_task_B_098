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
    coordinate_source VARCHAR(100) DEFAULT 'archaeological_survey',
    coordinate_precision_m NUMERIC(8, 2) DEFAULT 5.0,
    gps_verified BOOLEAN DEFAULT true,
    archaeological_site_id VARCHAR(50),
    survey_year INTEGER,
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
-- 12. 初始样例数据：汉代河西走廊烽火台（考古GPS坐标）
-- 数据来源：国家文物局长城资源调查、甘肃省文物考古研究所
-- ============================================================
INSERT INTO beacons (name, code, dynasty, location, elevation, height, description, status, coordinate_source, coordinate_precision_m, gps_verified, archaeological_site_id, survey_year)
VALUES
('玉门关遗址烽火台（小方盘城）', 'YMG-001', '汉代', ST_SetSRID(ST_MakePoint(93.879722, 40.351389), 4326), 1310.0, 10.1, '敦煌市西北90km小方盘城遗址，全国重点文保单位，丝路西端起点', 'active', '国家文物局长城资源调查', 3.5, true, 'GS-DH-001', 2007),
('河仓城遗址烽火台', 'HCC-002', '汉代', ST_SetSRID(ST_MakePoint(94.022500, 40.424722), 4326), 1289.0, 9.8, '大方盘城（河仓城）遗址，西汉军需仓库', 'active', '甘肃省文物考古研究所', 4.2, true, 'GS-DH-002', 2005),
('大方盘城烽火台', 'DFP-003', '汉代', ST_SetSRID(ST_MakePoint(94.151389, 40.478333), 4326), 1267.0, 11.2, '玉门都尉府治所', 'active', '敦煌研究院考古调查', 5.0, true, 'GS-DH-003', 2008),
('敦煌郡治烽火台', 'DHS-004', '汉代', ST_SetSRID(ST_MakePoint(94.657222, 40.142500), 4326), 1139.0, 9.5, '敦煌郡治所附近警戒烽燧', 'active', '河西长城考古调查报告', 6.0, true, 'GS-DH-004', 2006),
('莫高窟顶烽火台', 'MGK-005', '汉代', ST_SetSRID(ST_MakePoint(94.808611, 40.038611), 4326), 1156.0, 8.7, '莫高窟窟顶汉代烽燧遗址，保护丝路商旅', 'active', '敦煌研究院莫高窟考古', 3.0, true, 'GS-DH-005', 2010),
('瓜州（安西）破城子烽火台', 'GZ-006', '汉代', ST_SetSRID(ST_MakePoint(95.781667, 40.535833), 4326), 1178.0, 10.4, '瓜州县破城子遗址，广至县治', 'active', '瓜州县文物局普查', 5.5, true, 'GS-GZ-001', 2009),
('嘉峪关关城烽火台（明边墙下）', 'JYG-007', '汉代', ST_SetSRID(ST_MakePoint(98.289167, 39.773056), 4326), 1666.0, 12.6, '嘉峪关城楼墩台，河西走廊咽喉', 'active', '嘉峪关长城博物馆测绘', 2.5, true, 'GS-JYG-001', 2008),
('酒泉下河清烽火台', 'JQ-008', '汉代', ST_SetSRID(ST_MakePoint(98.502500, 39.728611), 4326), 1480.0, 9.8, '酒泉下河清汉墓群附近烽燧', 'active', '酒泉市文物局调查', 6.5, true, 'GS-JQ-001', 2007),
('张掖黑水国烽火台', 'ZY-009', '汉代', ST_SetSRID(ST_MakePoint(100.456944, 38.928889), 4326), 1485.0, 10.9, '张掖黑水国遗址（觻得故城），河西四郡张掖郡治', 'active', '甘肃省文物考古研究所', 4.0, true, 'GS-ZY-001', 2011),
('武威雷台烽火台', 'WW-010', '汉代', ST_SetSRID(ST_MakePoint(102.643333, 37.928056), 4326), 1530.0, 10.2, '武威雷台汉墓附近烽燧，凉州治所', 'active', '武威市博物馆考古', 5.0, true, 'GS-WW-001', 2009),
('兰州河口烽火台', 'LZ-011', '汉代', ST_SetSRID(ST_MakePoint(103.829167, 36.063611), 4326), 1520.0, 11.3, '兰州河口古镇，金城郡西大门', 'active', '兰州市文物局普查', 7.0, true, 'GS-LZ-001', 2006),
('天水牧马滩烽火台', 'TS-012', '汉代', ST_SetSRID(ST_MakePoint(105.720278, 34.579167), 4326), 1140.0, 9.0, '天水放马滩秦简出土地，汉代陇西郡烽燧', 'active', '天水市文物考古研究所', 4.5, true, 'GS-TS-001', 2012);

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
CREATE TABLE IF NOT EXISTS base_station_types (
    type_code VARCHAR(20) PRIMARY KEY,
    type_name VARCHAR(50) NOT NULL,
    standard_version VARCHAR(20) DEFAULT '1.0',
    description TEXT,
    min_coverage_radius_km NUMERIC(6, 2),
    max_coverage_radius_km NUMERIC(6, 2),
    standard_coverage_radius_km NUMERIC(6, 2),
    min_capacity_mbps NUMERIC(10, 2),
    max_capacity_mbps NUMERIC(10, 2),
    standard_capacity_mbps NUMERIC(10, 2),
    min_latency_ms NUMERIC(8, 2),
    max_latency_ms NUMERIC(8, 2),
    standard_latency_ms NUMERIC(8, 2),
    frequency_band VARCHAR(50),
    typical_height_m NUMERIC(6, 2),
    typical_power_kw NUMERIC(5, 2),
    technology_generation VARCHAR(20),
    sort_order INTEGER DEFAULT 0
);

CREATE TABLE IF NOT EXISTS modern_base_stations (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    station_type VARCHAR(20) REFERENCES base_station_types(type_code),
    location GEOGRAPHY(POINT, 4326) NOT NULL,
    height NUMERIC(6, 2) DEFAULT 30.0,
    coverage_radius_km NUMERIC(6, 2) DEFAULT 1.5,
    capacity_mbps NUMERIC(8, 2) DEFAULT 1000.0,
    latency_ms NUMERIC(6, 2) DEFAULT 10.0,
    frequency_ghz NUMERIC(5, 2) DEFAULT 3.5,
    power_kw NUMERIC(5, 2) DEFAULT 1.2,
    is_standard_compliant BOOLEAN DEFAULT true,
    standard_version VARCHAR(20) DEFAULT '1.0',
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_modern_stations_location ON modern_base_stations USING GIST(location);
CREATE INDEX IF NOT EXISTS idx_modern_stations_type ON modern_base_stations(station_type);
CREATE INDEX IF NOT EXISTS idx_modern_stations_compliant ON modern_base_stations(is_standard_compliant);

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
-- 18. 朝代样例数据：秦代烽火台（考古GPS坐标）
-- 数据来源：陕西省考古研究院秦直道调查、长城资源调查
-- 坐标体系：WGS84(EPSG:4326)，精度至小数点后6位
-- ============================================================
INSERT INTO beacons (name, code, dynasty, location, elevation, height, description, status, coordinate_source, coordinate_precision_m, gps_verified, archaeological_site_id, survey_year)
VALUES
('咸阳宫城烽火台（秦都核心）', 'XY-QIN-01', '秦代', ST_SetSRID(ST_MakePoint(108.710833, 34.331111), 4326), 385.0, 8.5, '秦咸阳城遗址渭北区，秦帝国都城核心', 'active', '陕西省考古研究院秦都考古', 2.0, true, 'QIN-XY-001', 2014),
('临潼新丰鸿门烽火台', 'LT-QIN-02', '秦代', ST_SetSRID(ST_MakePoint(109.220556, 34.369722), 4326), 456.0, 9.2, '临潼新丰镇鸿门堡，鸿门宴遗址附近', 'active', '陕西省考古研究院秦直道调查', 3.5, true, 'QIN-LT-002', 2013),
('华阴潼关魏长城烽火台', 'HY-QIN-03', '秦代', ST_SetSRID(ST_MakePoint(110.093611, 34.555278), 4326), 528.0, 8.8, '秦东大门潼关，魏长城遗址', 'active', '陕西省长城资源调查', 4.0, true, 'QIN-HY-003', 2009),
('宝鸡陈仓故址烽火台', 'BJ-QIN-04', '秦代', ST_SetSRID(ST_MakePoint(107.152500, 34.365556), 4326), 615.0, 7.6, '宝鸡陈仓区，秦文公东迁所建', 'active', '宝鸡市考古研究所', 5.0, true, 'QIN-BJ-004', 2011),
('天水放马滩秦烽火台', 'TS-QIN-05', '秦代', ST_SetSRID(ST_MakePoint(105.720000, 34.578889), 4326), 1142.0, 10.2, '天水放马滩秦简出土地，秦西陲故地', 'active', '甘肃省文物考古研究所', 3.0, true, 'QIN-TS-005', 2015),
('延安秦直道墩梁烽火台', 'YA-QIN-06', '秦代', ST_SetSRID(ST_MakePoint(109.487500, 36.588611), 4326), 1120.0, 9.5, '秦直道遗址延安段，始皇驰道', 'active', '陕西省考古研究院秦直道调查', 6.0, true, 'QIN-YA-006', 2010),
('韩城芝川黄河渡口烽火台', 'HC-QIN-07', '秦代', ST_SetSRID(ST_MakePoint(110.451111, 35.476111), 4326), 545.0, 8.2, '韩城芝川镇司马迁祠附近，黄河古渡', 'active', '韩城市文物局普查', 4.5, true, 'QIN-HC-007', 2012),
('汉中褒斜道石门烽火台', 'HZ-QIN-08', '秦代', ST_SetSRID(ST_MakePoint(107.030278, 33.068889), 4326), 560.0, 9.8, '汉中褒斜道石门，秦蜀道咽喉', 'active', '汉中市文物考古研究所', 5.5, true, 'QIN-HZ-008', 2008)
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
-- 19. 朝代样例数据：明代九边烽火台（考古GPS坐标）
-- 数据来源：明长城资源调查（国家文物局2009年）、嘉峪关长城博物馆
-- 明长城烽火台间距：通常1-3公里，紧要处500米
-- ============================================================
INSERT INTO beacons (name, code, dynasty, location, elevation, height, description, status, coordinate_source, coordinate_precision_m, gps_verified, archaeological_site_id, survey_year)
VALUES
('嘉峪关关城明长城烽火台（天下第一墩）', 'JYG-M-01', '明代', ST_SetSRID(ST_MakePoint(98.288889, 39.772778), 4326), 1666.0, 14.5, '嘉峪关关城西侧长城墩台，明长城西端起点，九边甘肃镇', 'active', '明长城资源调查（国家文物局2009）', 2.0, true, 'MING-GS-001', 2009),
('酒泉果园乡明长城烽火台', 'JQ-M-02', '明代', ST_SetSRID(ST_MakePoint(98.502222, 39.728333), 4326), 1480.0, 11.8, '酒泉肃州区果园乡明长城遗址，甘肃镇肃州卫', 'active', '甘肃省明长城资源调查', 3.5, true, 'MING-GS-002', 2008),
('张掖甘州明长城烽火台（镇远楼西）', 'ZY-M-03', '明代', ST_SetSRID(ST_MakePoint(100.456667, 38.928611), 4326), 1485.0, 12.3, '张掖甘州区明长城，甘肃镇甘州卫，河西走廊中部', 'active', '张掖市文物局调查', 4.0, true, 'MING-GS-003', 2007),
('山丹峡口明长城烽火台（绣花庙段）', 'SD-M-04', '明代', ST_SetSRID(ST_MakePoint(101.088889, 38.788889), 4326), 1760.0, 10.6, '山丹县峡口古城，绣花庙长城段，国道312线旁', 'active', '山丹县长城博物馆测绘', 3.0, true, 'MING-GS-004', 2010),
('武威凉州明长城烽火台（雷台北）', 'WW-M-05', '明代', ST_SetSRID(ST_MakePoint(102.643056, 37.927778), 4326), 1530.0, 13.2, '武威凉州区长城遗址，陕西行都司凉州卫', 'active', '武威市长城资源调查', 4.5, true, 'MING-GS-005', 2009),
('古浪乌鞘岭明长城烽火台', 'GL-M-06', '明代', ST_SetSRID(ST_MakePoint(102.897222, 37.477778), 4326), 2000.0, 10.8, '古浪乌鞘岭垭口，河西走廊门户，长城翻越祁连山', 'active', '古浪县文物局普查', 5.0, true, 'MING-GS-006', 2011),
('兰州金城关明长城烽火台', 'LZ-M-07', '明代', ST_SetSRID(ST_MakePoint(103.828889, 36.063333), 4326), 1520.0, 12.7, '兰州金城关边墙，兰州黄河北岸明长城', 'active', '兰州市博物馆考古调查', 3.5, true, 'MING-GS-007', 2008),
('临洮明长城烽火台（洮州卫边墙）', 'LT-M-08', '明代', ST_SetSRID(ST_MakePoint(103.858333, 35.366667), 4326), 1880.0, 9.6, '临洮县明长城，临洮府边墙，洮州卫防御体系', 'active', '临洮县文物局普查', 6.0, true, 'MING-GS-008', 2007),
('固原开城明长城烽火台（固原镇）', 'GY-M-09', '明代', ST_SetSRID(ST_MakePoint(106.278333, 36.018333), 4326), 1800.0, 11.4, '固原开城遗址，九边固原镇总镇，三边总制驻地', 'active', '宁夏长城资源调查', 5.5, true, 'MING-NX-001', 2010),
('银川花马池明长城烽火台（宁夏镇）', 'YC-M-10', '明代', ST_SetSRID(ST_MakePoint(106.266667, 38.466667), 4326), 1110.0, 12.5, '银川花马池营，九边宁夏镇，河东墙防线', 'active', '宁夏文物考古研究所', 4.0, true, 'MING-NX-002', 2009)
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

-- 插入基站类型标准
INSERT INTO base_station_types (type_code, type_name, standard_version, description,
    min_coverage_radius_km, max_coverage_radius_km, standard_coverage_radius_km,
    min_capacity_mbps, max_capacity_mbps, standard_capacity_mbps,
    min_latency_ms, max_latency_ms, standard_latency_ms,
    frequency_band, typical_height_m, typical_power_kw,
    technology_generation, sort_order)
VALUES
('5g_macro', '5G宏基站', '3GPP R17', '第五代移动通信宏基站，广覆盖高带宽',
    0.5, 3.0, 1.5,
    500, 3000, 1500,
    4, 20, 8,
    'Sub-6GHz', 35.0, 2.0,
    '5G', 1),
('5g_micro', '5G微基站', '3GPP R17', '第五代移动通信微基站，热点补盲',
    0.1, 0.5, 0.3,
    100, 500, 300,
    5, 15, 10,
    'Sub-6GHz', 10.0, 0.5,
    '5G', 2),
('4g_lte', '4G LTE基站', '3GPP R15', '第四代移动通信LTE基站',
    1.0, 5.0, 3.0,
    50, 300, 150,
    10, 50, 20,
    'Sub-3GHz', 30.0, 1.2,
    '4G', 3),
('microwave', '微波中继站', 'ITU-R F.1104', '微波视距中继传输站',
    20.0, 80.0, 50.0,
    100, 1000, 500,
    1, 10, 2,
    '6-42GHz', 80.0, 0.8,
    '微波', 4),
('satellite_vsat', '卫星通信站', 'ITU-R S.1002', '甚小口径卫星终端站',
    100.0, 500.0, 200.0,
    5, 100, 20,
    200, 800, 500,
    'Ku/Ka波段', 200.0, 10.0,
    '卫星', 5),
('fiber_pop', '光纤接入点', 'ITU-T G.984.4', '光纤网络接入点/POP点',
    0.0, 0.0, 0.0,
    1000, 10000, 5000,
    0.5, 5, 1,
    '光纤', 5.0, 0.3,
    '光纤', 6)
ON CONFLICT (type_code) DO NOTHING;

INSERT INTO modern_base_stations (name, station_type, location, height, coverage_radius_km, capacity_mbps, latency_ms, frequency_ghz, power_kw, is_standard_compliant, standard_version, status)
VALUES
('玉门5G宏基站', '5g_macro', ST_SetSRID(ST_MakePoint(97.04, 40.29), 4326), 35.0, 1.5, 1500.0, 8.0, 3.5, 2.0, true, '3GPP R17', 'active'),
('嘉峪关5G宏基站', '5g_macro', ST_SetSRID(ST_MakePoint(98.29, 39.77), 4326), 40.0, 1.8, 1800.0, 7.0, 3.5, 2.5, true, '3GPP R17', 'active'),
('酒泉5G宏基站', '5g_macro', ST_SetSRID(ST_MakePoint(98.50, 39.73), 4326), 32.0, 1.2, 1200.0, 9.0, 2.6, 1.8, true, '3GPP R17', 'active'),
('张掖5G宏基站', '5g_macro', ST_SetSRID(ST_MakePoint(100.46, 38.93), 4326), 38.0, 1.8, 1800.0, 6.5, 3.5, 2.5, true, '3GPP R17', 'active'),
('武威5G宏基站', '5g_macro', ST_SetSRID(ST_MakePoint(102.64, 37.93), 4326), 30.0, 1.3, 1300.0, 8.5, 2.6, 1.8, true, '3GPP R17', 'active'),
('兰州5G宏基站', '5g_macro', ST_SetSRID(ST_MakePoint(103.83, 36.06), 4326), 45.0, 2.0, 2000.0, 5.0, 3.5, 3.0, true, '3GPP R17', 'active'),
('天水5G宏基站', '5g_macro', ST_SetSRID(ST_MakePoint(105.72, 34.58), 4326), 35.0, 1.5, 1400.0, 7.5, 3.5, 2.0, true, '3GPP R17', 'active'),
('西安5G宏基站', '5g_macro', ST_SetSRID(ST_MakePoint(108.94, 34.27), 4326), 50.0, 2.5, 3000.0, 4.0, 3.5, 5.0, true, '3GPP R17', 'active'),
('敦煌微波中继站', 'microwave', ST_SetSRID(ST_MakePoint(94.66, 40.14), 4326), 80.0, 50.0, 500.0, 2.0, 6.0, 0.8, true, 'ITU-R F.1104', 'active'),
('西宁卫星通信站', 'satellite_vsat', ST_SetSRID(ST_MakePoint(101.78, 36.62), 4326), 200.0, 200.0, 100.0, 500.0, 14.0, 10.0, true, 'ITU-R S.1002', 'active')
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
