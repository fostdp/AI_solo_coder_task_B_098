#!/usr/bin/env python3
# -*- coding: utf-8 -*-

import json
import time
import random
import math
import requests
import argparse
import os
import signal as sig
from datetime import datetime

API_BASE = os.environ.get("API_BASE", "http://localhost:8080/api")

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

WEATHER_PROFILES = {
    "clear": {
        "name": "晴朗",
        "visibility_range": (15.0, 23.0),
        "wind_range": (3.0, 7.0),
        "factor": 1.0,
    },
    "light_haze": {
        "name": "轻雾",
        "visibility_range": (8.0, 12.0),
        "wind_range": (2.0, 5.0),
        "factor": 0.8,
    },
    "foggy": {
        "name": "大雾",
        "visibility_range": (3.0, 6.0),
        "wind_range": (1.0, 3.0),
        "factor": 0.6,
    },
    "heavy_fog": {
        "name": "浓雾",
        "visibility_range": (1.0, 2.5),
        "wind_range": (0.5, 2.0),
        "factor": 0.4,
    },
    "sandstorm": {
        "name": "沙尘暴",
        "visibility_range": (0.3, 1.0),
        "wind_range": (12.0, 18.0),
        "factor": 0.2,
    },
}

class BeaconSensorSimulator:
    def __init__(self, beacons=BEACON_BASE_DATA, interval=60,
                 weather="auto", fixed_visibility=None):
        self.beacons = beacons
        self.interval = interval
        self.weather_mode = weather
        self.fixed_visibility = fixed_visibility
        self.weather_pattern = "clear"
        self.weather_change_counter = 0
        self.beacon_states = {}
        self.running = True
        self.cycle_count = 0

        for beacon in beacons:
            profile = WEATHER_PROFILES.get(self.weather_pattern, WEATHER_PROFILES["clear"])
            vis_min, vis_max = profile["visibility_range"]
            wind_min, wind_max = profile["wind_range"]
            self.beacon_states[beacon["id"]] = {
                "visibility": vis_min + random.random() * (vis_max - vis_min),
                "wind_speed": wind_min + random.random() * (wind_max - wind_min),
                "wind_direction": random.random() * 360,
                "temperature": 15.0 + random.random() * 5.0,
                "humidity": 40.0 + random.random() * 20.0,
            }

        sig.signal(sig.SIGINT, self._signal_handler)
        sig.signal(sig.SIGTERM, self._signal_handler)

    def _signal_handler(self, signum, frame):
        print(f"\n收到停止信号，正在优雅退出...")
        self.running = False

    def update_weather(self):
        if self.weather_mode != "auto":
            self.weather_pattern = self.weather_mode
            return

        self.weather_change_counter += 1
        if self.weather_change_counter >= random.randint(5, 15):
            self.weather_change_counter = 0
            patterns = [
                "clear", "clear", "clear",
                "light_haze", "light_haze",
                "foggy",
                "heavy_fog",
                "sandstorm"
            ]
            self.weather_pattern = random.choice(patterns)
            profile = WEATHER_PROFILES[self.weather_pattern]
            print(f"[{datetime.now().strftime('%H:%M:%S')}] 天气变化: {profile['name']} ({self.weather_pattern})")

    def get_weather_factor(self):
        if self.fixed_visibility is not None:
            if self.fixed_visibility >= 10:
                return 1.0
            elif self.fixed_visibility >= 5:
                return 0.8
            elif self.fixed_visibility >= 2:
                return 0.6
            elif self.fixed_visibility >= 1:
                return 0.4
            else:
                return 0.2
        profile = WEATHER_PROFILES.get(self.weather_pattern, WEATHER_PROFILES["clear"])
        return profile["factor"]

    def generate_sensor_data(self, beacon_id):
        state = self.beacon_states[beacon_id]
        beacon_info = next((b for b in self.beacons if b["id"] == beacon_id), None)
        if not beacon_info:
            return None

        weather_factor = self.get_weather_factor()

        if self.fixed_visibility is not None:
            target_visibility = self.fixed_visibility + random.gauss(0, 0.3)
            target_visibility = max(0.1, target_visibility)
        else:
            profile = WEATHER_PROFILES.get(self.weather_pattern, WEATHER_PROFILES["clear"])
            vis_min, vis_max = profile["visibility_range"]
            target_visibility = vis_min + random.random() * (vis_max - vis_min)

        state["visibility"] += (target_visibility - state["visibility"]) * 0.3
        state["visibility"] += random.gauss(0, 0.5)
        state["visibility"] = max(0.1, min(30.0, state["visibility"]))

        if self.fixed_visibility is None:
            profile = WEATHER_PROFILES.get(self.weather_pattern, WEATHER_PROFILES["clear"])
            wind_min, wind_max = profile["wind_range"]
            base_wind = wind_min + random.random() * (wind_max - wind_min)
            state["wind_speed"] += (base_wind - state["wind_speed"]) * 0.2
            state["wind_speed"] += random.gauss(0, 0.3)
            state["wind_speed"] = max(0, min(20.0, state["wind_speed"]))
        else:
            state["wind_speed"] += random.gauss(0, 0.2)
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
        except requests.RequestException:
            return False

    def send_signal_reception(self, data):
        try:
            response = requests.post(
                f"{API_BASE}/signal-reception",
                json=data,
                timeout=5
            )
            return response.status_code == 201
        except requests.RequestException:
            return False

    def run_cycle(self):
        self.update_weather()
        self.cycle_count += 1

        profile = WEATHER_PROFILES.get(self.weather_pattern, WEATHER_PROFILES["clear"])
        weather_name = profile["name"]
        vis_info = f"固定能见度={self.fixed_visibility}km" if self.fixed_visibility else f"天气={weather_name}"
        print(f"\n{'='*60}")
        print(f"[{datetime.now().strftime('%Y-%m-%d %H:%M:%S')}] 周期 #{self.cycle_count}")
        print(f"模式: {vis_info} (因子: {self.get_weather_factor():.2f})")
        print(f"{'='*60}")

        success_count = 0
        for beacon in self.beacons:
            sensor_data = self.generate_sensor_data(beacon["id"])
            if sensor_data:
                if self.send_sensor_data(sensor_data):
                    success_count += 1
                    print(f"  + {beacon['name']} - 能见度:{sensor_data['visibility']:.1f}km "
                          f"风速:{sensor_data['wind_speed']:.1f}m/s "
                          f"温度:{sensor_data['temperature']:.1f}C")
                else:
                    print(f"  x {beacon['name']} - 数据发送失败")

        print(f"\n--- 信号接收状态 ---")
        signal_success = 0
        for from_id, to_id in BEACON_LINKS:
            reception = self.generate_signal_reception(from_id, to_id)
            if reception:
                if self.send_signal_reception(reception):
                    signal_success += 1
                    from_name = next((b['name'] for b in self.beacons if b['id'] == from_id), '')
                    to_name = next((b['name'] for b in self.beacons if b['id'] == to_id), '')
                    status = "+" if reception['is_received'] else "x"
                    print(f"  {status} {from_name[:4]} -> {to_name[:4]} "
                          f"信号:{reception['signal_strength']:.1f}")

        print(f"\n周期完成: {success_count}/{len(self.beacons)} 传感器, "
              f"{signal_success}/{len(BEACON_LINKS)} 链路")

    def run(self):
        fixed_vis_str = f" 固定能见度={self.fixed_visibility}km" if self.fixed_visibility else ""
        print("=" * 60)
        print("  古代烽火台传感器模拟器启动")
        print(f"  烽火台数量: {len(self.beacons)}")
        print(f"  上报间隔: {self.interval}秒")
        print(f"  API地址: {API_BASE}")
        print(f"  天气模式: {self.weather_mode}{fixed_vis_str}")
        print("=" * 60)

        while self.running:
            self.run_cycle()
            print(f"\n下一次上报: {self.interval}秒后...")
            for _ in range(self.interval):
                if not self.running:
                    break
                time.sleep(1)

        print("\n模拟器已停止")


def main():
    parser = argparse.ArgumentParser(
        description="古代烽火台传感器模拟器",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
天气模式:
  auto        自动随机切换天气 (默认)
  clear       晴朗 - 能见度15-23km, 微风
  light_haze  轻雾 - 能见度8-12km
  foggy       大雾 - 能见度3-6km
  heavy_fog   浓雾 - 能见度1-2.5km
  sandstorm   沙尘暴 - 能见度0.3-1km, 强风

示例:
  python beacon_simulator.py                          # 默认60秒间隔, 自动天气
  python beacon_simulator.py --interval 30            # 30秒间隔
  python beacon_simulator.py --weather sandstorm      # 固定沙尘暴
  python beacon_simulator.py --visibility 2.0         # 固定能见度2km
  python beacon_simulator.py --weather foggy -i 10    # 大雾模式10秒快速上报
  python beacon_simulator.py --test                   # 单次测试
        """
    )

    parser.add_argument("-i", "--interval", type=int, default=60,
                        help="传感器上报间隔(秒), 默认60")
    parser.add_argument("-w", "--weather", type=str, default="auto",
                        choices=["auto", "clear", "light_haze", "foggy", "heavy_fog", "sandstorm"],
                        help="天气模式, 默认auto")
    parser.add_argument("-v", "--visibility", type=float, default=None,
                        help="固定能见度(km), 覆盖天气模式的能见度")
    parser.add_argument("--api", type=str, default=None,
                        help="API地址, 默认环境变量API_BASE或http://localhost:8080/api")
    parser.add_argument("--test", action="store_true",
                        help="运行单次测试后退出")

    args = parser.parse_args()

    global API_BASE
    if args.api:
        API_BASE = args.api

    if args.test:
        print("运行单次测试...")
        simulator = BeaconSensorSimulator(
            weather=args.weather,
            fixed_visibility=args.visibility
        )
        simulator.run_cycle()
        return

    simulator = BeaconSensorSimulator(
        interval=args.interval,
        weather=args.weather,
        fixed_visibility=args.visibility
    )
    simulator.run()


if __name__ == "__main__":
    main()
