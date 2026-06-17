class AdvancedFeatures {
    constructor(mapInstance, beaconMap) {
        this.map = mapInstance;
        this.beaconMap = beaconMap;
        this.modernStations = [];
        this.modernMarkers = [];
        this.showModernStations = false;
        this.currentTopologyId = 1;
        this.currentDynasty = 'han';
        this.ignitionPath = [];
        this.isAnimating = false;

        this.init();
    }

    init() {
        this.initPanelToggles();
        this.initDynastyComparison();
        this.initCrossEraComparison();
        this.initResilienceAnalysis();
        this.initIgnition();
        this.initDynastySwitch();
    }

    initPanelToggles() {
        const toggles = document.querySelectorAll('.panel-toggle');
        toggles.forEach(toggle => {
            toggle.addEventListener('click', () => {
                const targetId = toggle.getAttribute('data-toggle');
                const content = document.getElementById(targetId);
                const icon = toggle.querySelector('.toggle-icon');

                if (content.classList.contains('collapsed')) {
                    content.classList.remove('collapsed');
                    icon.style.transform = 'rotate(0deg)';
                } else {
                    content.classList.add('collapsed');
                    icon.style.transform = 'rotate(-90deg)';
                }
            });
        });
    }

    initDynastyComparison() {
        const btn = document.getElementById('compare-dynasties-btn');
        if (btn) {
            btn.addEventListener('click', () => this.compareDynasties());
        }
    }

    async compareDynasties() {
        const checkboxes = document.querySelectorAll('#dynasty-panel input[type="checkbox"]:checked');
        const dynasties = Array.from(checkboxes).map(cb => cb.value);

        if (dynasties.length === 0) {
            alert('请至少选择一个朝代');
            return;
        }

        try {
            const response = await fetch(`/api/dynasty-comparison?dynasties=${dynasties.join(',')}`);
            const data = await response.json();
            this.renderDynastyResults(data.comparisons);
        } catch (error) {
            console.error('朝代对比失败:', error);
            document.getElementById('dynasty-results').innerHTML =
                '<p class="text-muted">加载失败，请重试</p>';
        }
    }

    renderDynastyResults(comparisons) {
        const container = document.getElementById('dynasty-results');
        if (!container || !comparisons || comparisons.length === 0) {
            return;
        }
        let html = '';
        comparisons.forEach(comp => {
            html += `
                <div class="dynasty-card" style="border-left-color: ${comp.color}">
                    <h4 style="color: ${comp.color}">${comp.dynasty_name}</h4>
                    <div class="metrics-grid">
                        <div><span class="metric-label">烽火台数</span></div>
                        <div>${comp.node_count}</div>
                        <div><span class="metric-label">链路数</span></div>
                        <div>${comp.link_count}</div>
                        <div><span class="metric-label">连通度</span></div>
                        <div>${(comp.connectivity_index * 100).toFixed(1)}%</div>
                        <div><span class="metric-label">平均路径</span></div>
                        <div>${comp.avg_path_length ? comp.avg_path_length.toFixed(1) + ' 跳' : '--'}</div>
                        <div><span class="metric-label">网络直径</span></div>
                        <div>${comp.diameter} 跳</div>
                        <div><span class="metric-label">可靠性</span></div>
                        <div>${(comp.reliability * 100).toFixed(1)}%</div>
                        <div><span class="metric-label">网络密度</span></div>
                        <div>${(comp.density * 100).toFixed(1)}%</div>
                    </div>
                </div>
            `;
        });
        container.innerHTML = html;
    }

    initCrossEraComparison() {
        const btn = document.getElementById('cross-era-btn');
        if (btn) {
            btn.addEventListener('click', () => this.crossEraComparison());
        }

        const showBtn = document.getElementById('show-modern-stations-btn');
        if (showBtn) {
            showBtn.addEventListener('click', () => this.toggleModernStations());
        }
    }

    async crossEraComparison() {
        const topologyId = document.getElementById('ancient-topology-select').value || 1;

        try {
            const response = await fetch(`/api/cross-era-comparison?topology_id=${topologyId}`);
            const data = await response.json();
            this.renderCrossEraResults(data);
        } catch (error) {
            console.error('跨时代对比失败:', error);
            document.getElementById('cross-era-results').innerHTML =
                '<p class="text-muted">加载失败，请重试</p>';
        }
    }

    renderCrossEraResults(data) {
        const container = document.getElementById('cross-era-results');
        if (!container) return;

        const beacon = data.beacon_network || {};
        const modern = data.modern_network || {};
        const comp = data.comparison || {};

        container.innerHTML = `
            <table class="era-comparison-table">
                <tr>
                    <th>指标</th>
                    <th>古代烽火台</th>
                    <th>现代基站</th>
                </tr>
                <tr>
                    <td>节点数量</td>
                    <td>${beacon.node_count || '--'}</td>
                    <td class="highlight">${modern.node_count || '--'}</td>
                </tr>
                <tr>
                    <td>覆盖面积 (km²)</td>
                    <td>${beacon.total_coverage_km2 ? beacon.total_coverage_km2.toFixed(0) : '--'}</td>
                    <td class="highlight">${modern.total_coverage_km2 ? modern.total_coverage_km2.toFixed(0) : '--'}</td>
                </tr>
                <tr>
                    <td>总容量 (Mbps)</td>
                    <td>${beacon.total_capacity_mbps ? beacon.total_capacity_mbps.toFixed(3) : '--'}</td>
                    <td class="highlight">${modern.total_capacity_mbps ? modern.total_capacity_mbps.toFixed(0) : '--'}</td>
                </tr>
                <tr>
                    <td>平均延迟 (ms)</td>
                    <td>${beacon.avg_latency_ms ? beacon.avg_latency_ms.toFixed(0) : '--'}</td>
                    <td class="highlight">${modern.avg_latency_ms || '--'}</td>
                </tr>
                <tr>
                    <td>总功耗 (kW)</td>
                    <td>0 (柴火)</td>
                    <td>${modern.total_power_kw || '--'}</td>
                </tr>
                <tr>
                    <td>传输介质</td>
                    <td>可见光/烟</td>
                    <td>无线电波/微波</td>
                </tr>
            </table>
            <div style="margin-top: 10px; font-size: 0.75rem; color: #94a3b8;">
                年代差距: ${comp.era_gap_years || 2200} 年
                <br>
                容量倍率: ${comp.capacity_ratio ? (comp.capacity_ratio / 1000).toFixed(0) + ' 千倍' : '--'}
            </div>
        `;
    }

    async toggleModernStations() {
        if (this.showModernStations) {
            this.clearModernStations();
            this.showModernStations = false;
            document.getElementById('show-modern-stations-btn').textContent = '显示现代基站';
            return;
        }

        try {
            const response = await fetch('/api/modern-stations');
            const data = await response.json();
            this.modernStations = data.stations || [];
            this.renderModernStations();
            this.showModernStations = true;
            document.getElementById('show-modern-stations-btn').textContent = '隐藏现代基站';
        } catch (error) {
            console.error('加载现代基站失败:', error);
        }
    }

    renderModernStations() {
        this.modernStations.forEach(station => {
            const icon = L.divIcon({
                className: 'modern-station-marker',
                iconSize: [12, 12],
                iconAnchor: [6, 6]
            });

            const marker = L.marker([station.lat, station.lon], { icon: icon })
                .addTo(this.map)
                .bindPopup(`
                    <h4>${station.name}</h4>
                    <div class="popup-info">
                        <div><span class="info-label">类型:</span> ${station.station_type}</div>
                        <div><span class="info-label">覆盖:</span> ${station.coverage_radius_km} km</div>
                        <div><span class="info-label">容量:</span> ${station.capacity_mbps} Mbps</div>
                        <div><span class="info-label">延迟:</span> ${station.latency_ms} ms</div>
                        <div><span class="info-label">频率:</span> ${station.frequency_ghz} GHz</div>
                    </div>
                `);

            const circle = L.circle([station.lat, station.lon], {
                color: '#3b82f6',
                fillColor: '#3b82f6',
                fillOpacity: 0.1,
                radius: station.coverage_radius_km * 1000,
                weight: 1
            }).addTo(this.map);

            this.modernMarkers.push(marker, circle);
        });
    }

    clearModernStations() {
        this.modernMarkers.forEach(m => {
            if (m && m.remove) m.remove();
        });
        this.modernMarkers = [];
    }

    initResilienceAnalysis() {
        const btn = document.getElementById('resilience-analyze-btn');
        if (btn) {
            btn.addEventListener('click', () => this.analyzeResilience());
        }

        const slider = document.getElementById('resilience-steps');
        const valueSpan = document.getElementById('resilience-steps-value');
        if (slider && valueSpan) {
            slider.addEventListener('input', () => {
                valueSpan.textContent = slider.value;
            });
        }
    }

    async analyzeResilience() {
        const attackType = document.getElementById('attack-type-select').value;
        const steps = parseInt(document.getElementById('resilience-steps').value);

        try {
            const response = await fetch('/api/resilience', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    topology_id: this.currentTopologyId,
                    attack_type: attackType,
                    steps: steps,
                    iterations: 1
                })
            });

            const data = await response.json();
            this.renderResilienceResults(data);
            this.drawResilienceChart(data);
        } catch (error) {
            console.error('抗毁性分析失败:', error);
        }
    }

    renderResilienceResults(data) {
        const container = document.getElementById('resilience-results');
        if (!container) return;

        container.innerHTML = `
            <div class="resilience-stat">
                <span>总节点数</span>
                <span>${data.total_nodes}</span>
            </div>
            <div class="resilience-stat">
                <span>鲁棒性得分</span>
                <span>${(data.robustness_score * 100).toFixed(1)}%</span>
            </div>
            <div class="resilience-stat">
                <span>临界崩溃阈值</span>
                <span>${(data.critical_threshold * 100).toFixed(1)}%</span>
            </div>
            <div class="resilience-stat">
                <span>攻击方式</span>
                <span>${this.getAttackTypeName(data.attack_type)}</span>
            </div>
        `;
    }

    getAttackTypeName(type) {
        const names = {
            'random': '随机攻击',
            'degree': '度优先攻击',
            'betweenness': '介数优先攻击',
            'critical': '关键节点攻击'
        };
        return names[type] || type;
    }

    drawResilienceChart(data) {
        const canvas = document.getElementById('resilience-chart');
        if (!canvas || !data.curve_points) return;

        const ctx = canvas.getContext('2d');
        const rect = canvas.getBoundingClientRect();
        canvas.width = rect.width * 2;
        canvas.height = rect.height * 2;
        ctx.scale(2, 2);

        const width = rect.width;
        const height = rect.height;
        const padding = { top: 10, right: 10, bottom: 20, left: 30 };
        const chartWidth = width - padding.left - padding.right;
        const chartHeight = height - padding.top - padding.bottom;

        ctx.clearRect(0, 0, width, height);

        ctx.strokeStyle = 'rgba(255, 255, 255, 0.1)';
        ctx.lineWidth = 1;
        for (let i = 0; i <= 4; i++) {
            const y = padding.top + (chartHeight / 4) * i;
            ctx.beginPath();
            ctx.moveTo(padding.left, y);
            ctx.lineTo(width - padding.right, y);
            ctx.stroke();
        }

        const points = data.curve_points;
        if (points.length < 2) return;

        ctx.beginPath();
        ctx.strokeStyle = '#e94560';
        ctx.lineWidth = 2;

        points.forEach((p, i) => {
            const x = padding.left + p.removal_ratio * chartWidth;
            const y = padding.top + (1 - p.connectivity_index) * chartHeight;
            if (i === 0) {
                ctx.moveTo(x, y);
            } else {
                ctx.lineTo(x, y);
            }
        });
        ctx.stroke();

        ctx.fillStyle = 'rgba(233, 69, 96, 0.2)';
        ctx.beginPath();
        points.forEach((p, i) => {
            const x = padding.left + p.removal_ratio * chartWidth;
            const y = padding.top + (1 - p.connectivity_index) * chartHeight;
            if (i === 0) {
                ctx.moveTo(x, y);
            } else {
                ctx.lineTo(x, y);
            }
        });
        ctx.lineTo(padding.left + chartWidth, padding.top + chartHeight);
        ctx.lineTo(padding.left, padding.top + chartHeight);
        ctx.closePath();
        ctx.fill();

        ctx.fillStyle = '#94a3b8';
        ctx.font = '10px sans-serif';
        ctx.textAlign = 'center';
        ctx.fillText('0%', padding.left, height - 5);
        ctx.fillText('100%', width - padding.right, height - 5);

        ctx.textAlign = 'right';
        ctx.fillText('1.0', padding.left - 5, padding.top + 5);
        ctx.fillText('0', padding.left - 5, height - padding.bottom);
    }

    initIgnition() {
        const btn = document.getElementById('ignite-btn');
        if (btn) {
            btn.addEventListener('click', () => this.igniteBeacon());
        }

        const select = document.getElementById('ignition-beacon-select');
        if (select && this.beaconMap && this.beaconMap.beacons) {
            this.beaconMap.beacons.forEach(beacon => {
                const option = document.createElement('option');
                option.value = beacon.id;
                option.textContent = beacon.name;
                select.appendChild(option);
            });
        }
    }

    async igniteBeacon() {
        if (this.isAnimating) {
            alert('正在播放动画中，请稍候...');
            return;
        }

        const beaconId = parseInt(document.getElementById('ignition-beacon-select').value);
        const weatherFactor = parseFloat(document.getElementById('ignition-weather').value);
        const topologyId = parseInt(document.getElementById('ignition-dynasty').value);

        if (!beaconId) {
            alert('请选择要点燃的烽火台');
            return;
        }

        try {
            const response = await fetch('/api/ignite', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    beacon_id: beaconId,
                    topology_id: topologyId,
                    weather_factor: weatherFactor,
                    session_id: this.getSessionId()
                })
            });

            const data = await response.json();
            this.ignitionPath = data.path || [];
            this.renderIgnitionResults(data);
            this.animateIgnition(data);
        } catch (error) {
            console.error('点燃失败:', error);
            alert('点燃失败，请重试');
        }
    }

    renderIgnitionResults(data) {
        const container = document.getElementById('ignition-results');
        if (!container) return;

        let pathHtml = '';
        if (data.path && data.path.length > 0) {
            const sorted = [...data.path].sort((a, b) => a.step - b.step);
            sorted.forEach((step, idx) => {
                pathHtml += `
                    <div class="ignition-step">
                        <div class="step-number">${step.step + 1}</div>
                        <div class="step-name">${step.beacon_name}</div>
                        <div class="step-time">${(step.delay_ms / 1000).toFixed(1)}s</div>
                    </div>
                `;
            });
        }

        container.innerHTML = `
            <div style="margin-bottom: 8px;">
                <strong>到达烽火台数: <span style="color: #4ade80;">${data.reached_count}</span> 座
            </div>
            <div style="margin-bottom: 8px;">
                总传播时间: <span style="color: #fbbf24;">${(data.total_time_ms / 1000).toFixed(1)}</span> 秒
            </div>
            <div class="ignition-path">
                ${pathHtml}
            </div>
        `;
    }

    animateIgnition(data) {
        if (!data.path || data.path.length === 0) return;

        this.isAnimating = true;
        const sorted = [...data.path].sort((a, b) => a.delay_ms - b.delay_ms);

        if (window.signalSystem) {
            sorted.forEach((step, idx) => {
                setTimeout(() => {
                    if (this.beaconMap && this.beaconMap.markers) {
                        const marker = this.beaconMap.markers[step.beacon_id];
                        if (marker) {
                            const icon = marker.getElement();
                            if (icon) {
                                icon.classList.add('beacon-ignited');
                                setTimeout(() => {
                                    icon.classList.remove('beacon-ignited');
                                }, 1000);
                            }
                        }
                    }

                    if (window.signalSystem && this.beaconMap && this.beaconMap.markers) {
                        const marker = this.beaconMap.markers[step.beacon_id];
                        if (marker) {
                            const latlng = marker.getLatLng();
                            window.signalSystem.createRipple(latlng.lat, latlng.lng, 1500 + idx * 500);
                        }
                    }

                    if (idx === sorted.length - 1) {
                        setTimeout(() => {
                            this.isAnimating = false;
                        }, 2000);
                    }
                }, step.delay_ms * 0.3);
            });
        }
    }

    initDynastySwitch() {
        const btn = document.getElementById('switch-dynasty-btn');
        if (btn) {
            btn.addEventListener('click', () => this.switchDynastyNetwork());
        }
    }

    async switchDynastyNetwork() {
        const select = document.getElementById('ignition-dynasty');
        if (!select) return;

        const topologyId = parseInt(select.value);
        this.currentTopologyId = topologyId;

        alert('切换朝代网络功能：此功能会重新加载烽火台和链路显示');
    }

    getSessionId() {
        let sessionId = localStorage.getItem('beacon_session_id');
        if (!sessionId) {
            sessionId = 'session_' + Math.random().toString(36).substr(2, 12);
            localStorage.setItem('beacon_session_id', sessionId);
        }
        return sessionId;
    }
}
