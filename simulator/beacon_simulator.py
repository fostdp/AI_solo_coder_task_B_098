#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
古代烽火台传感器模拟器
模拟每座烽火台每1分钟上报能见度、风速、地形高程、相邻烽火台信号接收状态
"""

import json
import time
import random
import math
import requests
from datetime import datetime

API_BASE = "http://localhost:8080/api"

BEACON_BASE_DATA = [
    {"id": 1, "name": "玉门关烽火台", "code": "YMG-001", "base_elevation": 1250.0},
    {"id": 2, "name": "河仓城烽火台", "code": "HCC-002", "base_elevation": 1235.0},
    {"id": 3, "name": "大方盘城烽火台", "code": "DFP-003", "base_elevation": 1220.0},
    {"id": 4, "name": "敦煌市烽火台", "code": "DHS-004", "base_elevation": 1139.0},
    {"id": 5, "name": "莫高窟烽火台", "code": "MGK-005", "base_elevation": 1150.0},
    {"id": 6, "name": "瓜州烽火台", "code": "GZ-006", "base_elevation": 1178.0},
    {"id": 7, "name": "嘉峪关烽火台", "code": "JYG-007", "base_elevation": 1666.0},
    {"id": 8, "name": "酒泉烽火台", "code": "JQ-008", "base_elevation": 1480.0},
    {"id": 9, "name": "张掖烽火台", "code": "ZY-009", "base_elevation": 1485.0},
    {"id": 10, "name": "武威烽火台", "code": "WW-010", "base_elevation": 1530.0},
    {"id": 11, "name": "兰州烽火台", "code": "LZ-011", "base_elevation": 1520.0},
    {"id": 12, "name": "天水烽火台", "code": "TS-012", "base_elevation": 1140.0},
]

BEACON_LINKS = [
    (1, 2), (2, 3), (3, 4), (4, 5), (5, 6),
    (6, 7), (7, 8), (8, 9), (9, 10), (10, 11), (11, 12)
]

class WeatherPattern:
    CLEAR = "clear"
    LIGHT_HAZE = "light_haze"
    FOGGY = "foggy"
    HEAVY_FOG = "heavy_fog"
    SANDSTORM = "sandstorm"

class BeaconSensorSimulator:
    def __init__(self, beacons=BEACON_BASE_DATA, interval=60):
        self.beacons = beacons
        self.interval = interval
        self.weather_pattern = WeatherPattern.CLEAR
        self.weather_change_counter = 0
        self.beacon_states = {}
        
        for beacon in beacons:
            self.beacon_states[beacon["id"]] = {
                "visibility": 15.0 + random.random() * 5.0,
                "wind_speed": 3.0 + random.random() * 3.0,
                "wind_direction": random.random() * 360,
                "temperature": 15.0 + random.random() * 5.0,
                "humidity": 40.0 + random.random() * 20.0,
            }

    def update_weather(self):
        self.weather_change_counter += 1
        if self.weather_change_counter >= random.randint(5, 15):
            self.weather_change_counter = 0
            patterns = [
                WeatherPattern.CLEAR, WeatherPattern.CLEAR, WeatherPattern.CLEAR,
                WeatherPattern.LIGHT_HAZE, WeatherPattern.LIGHT_HAZE,
                WeatherPattern.FOGGY,
                WeatherPattern.HEAVY_FOG,
                WeatherPattern.SANDSTORM
            ]
            self.weather_pattern = random.choice(patterns)
            print(f"[{datetime.now().strftime('%H:%M:%S')}] 天气变化: {self.weather_pattern}")

    def get_weather_factor(self):
        factors = {
            WeatherPattern.CLEAR: 1.0,
            WeatherPattern.LIGHT_HAZE: 0.8,
            WeatherPattern.FOGGY: 0.6,
            WeatherPattern.HEAVY_FOG: 0.4,
            WeatherPattern.SANDSTORM: 0.2,
        }
        return factors.get(self.weather_pattern, 1.0)

    def generate_sensor_data(self, beacon_id):
        state = self.beacon_states[beacon_id]
        beacon_info = next((b for b in self.beacons if b["id"] == beacon_id), None)
        if not beacon_info:
            return None

        weather_factor = self.get_weather_factor()

        target_visibility = self._base_visibility() * weather_factor
        state["visibility"] += (target_visibility - state["visibility"]) * 0.3
        state["visibility"] += random.gauss(0, 0.5)
        state["visibility"] = max(0.1, min(30.0, state["visibility"]))

        base_wind = self._base_wind_speed()
        state["wind_speed"] += (base_wind - state["wind_speed"]) * 0.2
        state["wind_speed"] += random.gauss(0, 0.3)
        state["wind_speed"] = max(0, min(20.0, state["wind_speed"]))

        state["wind_direction"] += random.gauss(0, 5)
        state["wind_direction"] = state["wind_direction"] % 360

        hour = datetime.now().hour
        temp_variation = 5 * math.sin((hour - 6) * math.pi / 12)
        state["temperature"] = 15 + temp_variation + random.gauss(0, 0.5)

        state["humidity"] += random.gauss(0, 1.0)
        state["humidity"] = max(10.0, min(80.0, state["humidity"]))

        terrain_elevation = beacon_info["base_elevation"] + random.gauss(0, 2.0)

        return {
            "beacon_id": beacon_id,
            "timestamp": datetime.now().isoformat(),
            "visibility": round(state["visibility"], 2),
            "wind_speed": round(state["wind_speed"], 2),
            "wind_direction": round(state["wind_direction"], 1),
            "temperature": round(state["temperature"], 1),
            "humidity": round(state["humidity"], 1),
            "terrain_elevation": round(terrain_elevation, 2),
        }

    def _base_visibility(self):
        if self.weather_pattern == WeatherPattern.CLEAR:
            return 15.0 + random.random() * 8.0
        elif self.weather_pattern == WeatherPattern.LIGHT_HAZE:
            return 8.0 + random.random() * 4.0
        elif self.weather_pattern == WeatherPattern.FOGGY:
            return 3.0 + random.random() * 3.0
        elif self.weather_pattern == WeatherPattern.HEAVY_FOG:
            return 1.0 + random.random() * 1.5
        elif self.weather_pattern == WeatherPattern.SANDSTORM:
            return 0.3 + random.random() * 0.7
        return 10.0

    def _base_wind_speed(self):
        if self.weather_pattern == WeatherPattern.SANDSTORM:
            return 12.0 + random.random() * 6.0
        elif self.weather_pattern == WeatherPattern.HEAVY_FOG:
            return 1.0 + random.random() * 2.0
        else:
            return 3.0 + random.random() * 4.0

    def generate_signal_reception(self, from_id, to_id):
        from_state = self.beacon_states.get(from_id)
        to_state = self.beacon_states.get(to_id)
        if not from_state or not to_state:
            return None

        avg_visibility = (from_state["visibility"] + to_state["visibility"]) / 2
        avg_wind = (from_state["wind_speed"] + to_state["wind_speed"]) / 2

        visibility_factor = min(1.0, avg_visibility / 10.0)
        wind_factor = max(0.5, 1.0 - avg_wind / 20.0)
        distance_factor = 0.95

        base_signal_strength = 80.0 * visibility_factor * wind_factor * distance_factor
        signal_strength = base_signal_strength + random.gauss(0, 5.0)
        signal_strength = max(0, min(100, signal_strength))

        reception_threshold = 30.0
        is_received = signal_strength >= reception_threshold

        interference_level = max(0, 100 - signal_strength) * random.uniform(0.8, 1.2)
        interference_level = min(100, interference_level)

        weather_factor = self.get_weather_factor()

        return {
            "from_beacon_id": from_id,
            "to_beacon_id": to_id,
            "timestamp": datetime.now().isoformat(),
            "signal_strength": round(signal_strength, 2),
            "is_received": is_received,
            "interference_level": round(interference_level, 2),
            "weather_factor": round(weather_factor, 4),
        }

    def send_sensor_data(self, data):
        try:
            response = requests.post(
                f"{API_BASE}/sensor-data",
                json=data,
                timeout=5
            )
            return response.status_code == 201
        except requests.RequestException as e:
            print(f"  发送传感器数据失败: {e}")
            return False

    def send_signal_reception(self, data):
        try:
            response = requests.post(
                f"{API_BASE}/signal-reception",
                json=data,
                timeout=5
            )
            return response.status_code == 201
        except requests.RequestException as e:
            print(f"  发送信号接收数据失败: {e}")
            return False

    def run_cycle(self):
        self.update_weather()
        print(f"\n{'='*60}")
        print(f"[{datetime.now().strftime('%Y-%m-%d %H:%M:%S')}] 开始新的传感周期")
        print(f"当前天气: {self.weather_pattern} (因子: {self.get_weather_factor():.2f})")
        print(f"{'='*60}")

        success_count = 0
        for beacon in self.beacons:
            sensor_data = self.generate_sensor_data(beacon["id"])
            if sensor_data:
                if self.send_sensor_data(sensor_data):
                    success_count += 1
                    print(f"  ✓ {beacon['name']} - 能见度:{sensor_data['visibility']:.1f}km "
                          f"风速:{sensor_data['wind_speed']:.1f}m/s "
                          f"温度:{sensor_data['temperature']:.1f}°C")
                else:
                    print(f"  ✗ {beacon['name']} - 数据发送失败")

        print(f"\n--- 信号接收状态 ---")
        signal_success = 0
        for from_id, to_id in BEACON_LINKS:
            reception = self.generate_signal_reception(from_id, to_id)
            if reception:
                if self.send_signal_reception(reception):
                    signal_success += 1
                    from_name = next((b['name'] for b in self.beacons if b['id'] == from_id), '')
                    to_name = next((b['name'] for b in self.beacons if b['id'] == to_id), '')
                    status = "✓ 接收" if reception['is_received'] else "✗ 未接收"
                    print(f"  {status} {from_name[:4]} → {to_name[:4]} "
                          f"信号强度:{reception['signal_strength']:.1f}")

        print(f"\n周期完成: {success_count}/{len(self.beacons)} 个传感器, "
              f"{signal_success}/{len(BEACON_LINKS)} 条链路")

    def run(self):
        print("=" * 60)
        print("  古代烽火台传感器模拟器启动")
        print(f"  烽火台数量: {len(self.beacons)}")
        print(f"  上报间隔: {self.interval}秒")
        print(f"  API地址: {API_BASE}")
        print("=" * 60)

        try:
            while True:
                self.run_cycle()
                print(f"\n下一次上报: {self.interval}秒后...")
                time.sleep(self.interval)
        except KeyboardInterrupt:
            print("\n\n模拟器已停止")

def run_single_test():
    print("运行单次测试...")
    simulator = BeaconSensorSimulator()
    simulator.run_cycle()

if __name__ == "__main__":
    import sys
    
    if len(sys.argv) > 1 and sys.argv[1] == "--test":
        run_single_test()
    else:
        interval = 60
        if len(sys.argv) > 1:
            try:
                interval = int(sys.argv[1])
            except ValueError:
                pass
        
        simulator = BeaconSensorSimulator(interval=interval)
        simulator.run()
