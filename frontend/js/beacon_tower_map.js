const API_BASE = 'http://localhost:8080/api';

let map;
let beaconMarkers = {};
let linkPolylines = [];
let viewshedPolygons = [];
let beaconsData = [];
let showLinks = true;
let showLabels = true;

function initMap() {
    map = L.map('map', {
        center: [38.5, 98.0],
        zoom: 6,
        zoomControl: true,
        attributionControl: false
    });

    L.tileLayer('https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png', {
        maxZoom: 19,
        subdomains: 'abcd'
    }).addTo(map);

    loadBeacons();
    setupCanvas();
    setupEventListeners();
}

function setupEventListeners() {
    window.addEventListener('resize', () => {
        resizeCanvas();
    });

    map.on('moveend', () => {
        drawSignals();
    });

    map.on('zoomend', () => {
        drawSignals();
    });
}

async function loadBeacons() {
    try {
        const response = await fetch(`${API_BASE}/beacons`);
        beaconsData = await response.json();

        beaconsData.forEach(beacon => {
            addBeaconMarker(beacon);
        });

        populateBeaconSelects();
        loadNetworkTopology();
        updateStats();

        document.getElementById('beacon-count').textContent = beaconsData.length;

    } catch (error) {
        console.error('Failed to load beacons:', error);
        showErrorToast('无法加载烽火台数据');
    }
}

function addBeaconMarker(beacon) {
    const isCritical = beacon.id === 1 || beacon.id === 7;

    const icon = L.divIcon({
        className: 'custom-beacon-icon',
        html: `
            <div class="beacon-marker ${isCritical ? 'critical' : ''}">
                <div class="beacon-pulse"></div>
                <div class="beacon-core"></div>
                ${showLabels ? `<div class="beacon-label">${beacon.name}</div>` : ''}
            </div>
        `,
        iconSize: [20, 20],
        iconAnchor: [10, 10]
    });

    const marker = L.marker([beacon.lat, beacon.lon], { icon: icon });

    const popupContent = `
        <div class="beacon-popup">
            <h4>🏯 ${beacon.name}</h4>
            <div class="popup-info">
                <p><span class="info-label">编号:</span> ${beacon.code}</p>
                <p><span class="info-label">朝代:</span> ${beacon.dynasty}</p>
                <p><span class="info-label">海拔:</span> ${beacon.elevation.toFixed(1)} 米</p>
                <p><span class="info-label">台高:</span> ${beacon.height} 米</p>
                <p><span class="info-label">坐标精度:</span> ${beacon.coordinate_precision_m?.toFixed(1) || '--'} 米</p>
                <p><span class="info-label">GPS验证:</span> ${beacon.gps_verified ? '✅ 已验证' : '❌ 未验证'}</p>
                <p><span class="info-label">考古编号:</span> ${beacon.archaeological_site_id || '--'}</p>
                <p><span class="info-label">调查年份:</span> ${beacon.survey_year || '--'}</p>
                <p><span class="info-label">状态:</span> ${beacon.status === 'active' ? '正常' : '停用'}</p>
            </div>
        </div>
    `;

    marker.bindPopup(popupContent);
    marker.on('click', () => {
        showBeaconDetails(beacon);
    });

    marker.addTo(map);
    beaconMarkers[beacon.id] = marker;
}

function populateBeaconSelects() {
    const fromSelect = document.getElementById('from-beacon-select');

    beaconsData.forEach(beacon => {
        const option = document.createElement('option');
        option.value = beacon.id;
        option.textContent = beacon.name;
        fromSelect.appendChild(option);
    });
}

async function loadNetworkTopology() {
    try {
        const response = await fetch(`${API_BASE}/network/topology`);
        const links = await response.json();

        drawLinks(links);
    } catch (error) {
        console.error('Failed to load network topology:', error);
    }
}

function drawLinks(links) {
    linkPolylines.forEach(line => map.removeLayer(line));
    linkPolylines = [];

    links.forEach(link => {
        const fromBeacon = beaconsData.find(b => b.id === link.from_beacon_id);
        const toBeacon = beaconsData.find(b => b.id === link.to_beacon_id);

        if (fromBeacon && toBeacon && showLinks) {
            const isCritical = link.is_critical;
            const color = isCritical ? '#e94560' : '#4ade80';
            const opacity = link.base_reliability;
            const weight = isCritical ? 3 : 2;

            const polyline = L.polyline(
                [[fromBeacon.lat, fromBeacon.lon], [toBeacon.lat, toBeacon.lon]],
                {
                    color: color,
                    weight: weight,
                    opacity: opacity * 0.8,
                    dashArray: isCritical ? null : '5, 5',
                    lineCap: 'round'
                }
            ).addTo(map);

            polyline.bindTooltip(`${fromBeacon.name} ↔ ${toBeacon.name}`, {
                permanent: false,
                direction: 'center'
            });

            linkPolylines.push(polyline);
        }
    });
}

function showBeaconDetails(beacon) {
    const linkInfo = document.getElementById('link-info');

    let linkedCount = 0;
    let html = `
        <h4 style="color: #e94560; margin-bottom: 10px;">${beacon.name}</h4>
        <div class="info-row">
            <span class="info-label">编号</span>
            <span class="info-value">${beacon.code}</span>
        </div>
        <div class="info-row">
            <span class="info-label">朝代</span>
            <span class="info-value">${beacon.dynasty}</span>
        </div>
        <div class="info-row">
            <span class="info-label">海拔</span>
            <span class="info-value">${beacon.elevation.toFixed(1)} m</span>
        </div>
        <div class="info-row">
            <span class="info-label">台高</span>
            <span class="info-value">${beacon.height} m</span>
        </div>
        <div class="info-row">
            <span class="info-label">经度</span>
            <span class="info-value">${beacon.lon.toFixed(4)}°</span>
        </div>
        <div class="info-row">
            <span class="info-label">纬度</span>
            <span class="info-value">${beacon.lat.toFixed(4)}°</span>
        </div>
        <div class="info-row">
            <span class="info-label">坐标精度</span>
            <span class="info-value">${beacon.coordinate_precision_m?.toFixed(1) || '--'} m</span>
        </div>
        <div class="info-row">
            <span class="info-label">GPS验证</span>
            <span class="info-value" style="color: ${beacon.gps_verified ? '#4ade80' : '#fbbf24'};">
                ${beacon.gps_verified ? '已验证' : '未验证'}
            </span>
        </div>
        <div class="info-row">
            <span class="info-label">考古编号</span>
            <span class="info-value">${beacon.archaeological_site_id || '--'}</span>
        </div>
        <div class="info-row">
            <span class="info-label">调查年份</span>
            <span class="info-value">${beacon.survey_year || '--'}</span>
        </div>
        <div class="info-row">
            <span class="info-label">数据来源</span>
            <span class="info-value" style="font-size: 11px;">${beacon.coordinate_source || '--'}</span>
        </div>
        <div class="info-row">
            <span class="info-label">状态</span>
            <span class="info-value" style="color: #4ade80;">正常</span>
        </div>
    `;

    linkInfo.innerHTML = html;
}

async function updateStats() {
    try {
        const connResponse = await fetch(`${API_BASE}/network/connectivity`);
        const connData = await connResponse.json();

        document.getElementById('connectivity-index').textContent =
            (connData.connectivity_index * 100).toFixed(1) + '%';

        if (connData.below_threshold) {
            document.getElementById('connectivity-index').style.color = '#ef4444';
        } else {
            document.getElementById('connectivity-index').style.color = '#4ade80';
        }

        const alertsResponse = await fetch(`${API_BASE}/alerts?resolved=false&limit=10`);
        const alertsData = await alertsResponse.json();

        const unresolved = alertsData.filter(a => !a.is_resolved).length;
        document.getElementById('alert-count').textContent = unresolved;

        updateAlertList(alertsData.slice(0, 5));

    } catch (error) {
        console.error('Failed to update stats:', error);
    }
}

function updateAlertList(alerts) {
    const alertList = document.getElementById('alert-list');

    if (alerts.length === 0) {
        alertList.innerHTML = '<p class="text-muted">暂无告警</p>';
        return;
    }

    let html = '';
    alerts.forEach(alert => {
        const severityClass = alert.severity === 'high' ? '' :
                             alert.severity === 'medium' ? 'warning' : 'info';
        const time = new Date(alert.created_at).toLocaleString('zh-CN', {
            month: 'short',
            day: 'numeric',
            hour: '2-digit',
            minute: '2-digit'
        });

        html += `
            <div class="alert-item ${severityClass}">
                <div class="alert-title">${alert.title}</div>
                <div class="alert-time">${time}</div>
            </div>
        `;
    });

    alertList.innerHTML = html;
}

function toggleLinks() {
    showLinks = !showLinks;
    const btn = document.getElementById('toggle-links-btn');
    btn.classList.toggle('active', showLinks);

    if (showLinks) {
        loadNetworkTopology();
    } else {
        linkPolylines.forEach(line => map.removeLayer(line));
        linkPolylines = [];
    }
}

function toggleLabels() {
    showLabels = !showLabels;
    const btn = document.getElementById('toggle-labels-btn');
    btn.classList.toggle('active', showLabels);

    Object.values(beaconMarkers).forEach(marker => map.removeLayer(marker));
    beaconMarkers = {};
    beaconsData.forEach(beacon => addBeaconMarker(beacon));
}

function showErrorToast(message) {
    console.error(message);
}

function updateCurrentTime() {
    const now = new Date();
    document.getElementById('current-time').textContent =
        now.toLocaleString('zh-CN', {
            year: 'numeric',
            month: '2-digit',
            day: '2-digit',
            hour: '2-digit',
            minute: '2-digit',
            second: '2-digit'
        });
}

document.addEventListener('DOMContentLoaded', () => {
    initMap();
    updateCurrentTime();
    setInterval(updateCurrentTime, 1000);
    setInterval(updateStats, 30000);
});
