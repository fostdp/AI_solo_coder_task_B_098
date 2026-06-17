let currentViewshed = null;

function setupVisibilityControls() {
    const showViewshedBtn = document.getElementById('show-viewshed-btn');
    const clearViewshedBtn = document.getElementById('clear-viewshed-btn');
    const distanceSlider = document.getElementById('viewshed-distance');
    const distanceValue = document.getElementById('viewshed-value');

    distanceSlider.addEventListener('input', () => {
        distanceValue.textContent = distanceSlider.value + ' km';
    });

    showViewshedBtn.addEventListener('click', showSelectedViewshed);
    clearViewshedBtn.addEventListener('click', clearViewshed);
}

async function showSelectedViewshed() {
    const beaconId = document.getElementById('from-beacon-select').value;
    const maxDistance = parseFloat(document.getElementById('viewshed-distance').value);

    if (!beaconId) {
        alert('请先选择一个烽火台');
        return;
    }

    try {
        const response = await fetch(
            `${API_BASE}/beacons/${beaconId}/viewshed?max_distance=${maxDistance}`
        );
        const data = await response.json();
        
        drawViewshedSector(data);
        highlightBeacon(parseInt(beaconId));
        
    } catch (error) {
        console.error('Failed to load viewshed:', error);
        drawLocalViewshed(parseInt(beaconId), maxDistance);
    }
}

function drawLocalViewshed(beaconId, maxDistanceKm) {
    const beacon = beaconsData.find(b => b.id === beaconId);
    if (!beacon) return;

    clearViewshed();

    const latLngs = generateSectorPoints(
        beacon.lat, beacon.lon, 0, 360, maxDistanceKm, 72
    );

    const polygon = L.polygon(latLngs, {
        color: '#e94560',
        weight: 2,
        opacity: 0.8,
        fillColor: '#e94560',
        fillOpacity: 0.15,
        dashArray: '10, 5'
    }).addTo(map);

    viewshedPolygons.push(polygon);
    currentViewshed = { beaconId, maxDistance: maxDistanceKm };
}

function drawViewshedSector(data) {
    clearViewshed();

    const latLngs = data.polygon.map(point => [point[1], point[0]]);

    const polygon = L.polygon(latLngs, {
        color: '#e94560',
        weight: 2,
        opacity: 0.8,
        fillColor: '#e94560',
        fillOpacity: 0.2,
        dashArray: '10, 5'
    }).addTo(map);

    viewshedPolygons.push(polygon);
    currentViewshed = { beaconId: data.beacon_id, data };

    const beacon = beaconsData.find(b => b.id === data.beacon_id);
    if (beacon) {
        map.panTo([beacon.lat, beacon.lon]);
    }
}

function clearViewshed() {
    viewshedPolygons.forEach(poly => map.removeLayer(poly));
    viewshedPolygons = [];
    currentViewshed = null;
    resetBeaconHighlights();
}

function generateSectorPoints(lat, lon, startAzimuth, endAzimuth, distanceKm, numPoints) {
    const points = [];
    points.push([lat, lon]);

    for (let i = 0; i <= numPoints; i++) {
        const azimuth = startAzimuth + (endAzimuth - startAzimuth) * (i / numPoints);
        const dest = destinationPoint(lat, lon, azimuth, distanceKm);
        points.push([dest.lat, dest.lon]);
    }

    points.push([lat, lon]);
    return points;
}

function destinationPoint(lat, lon, bearingDeg, distanceKm) {
    const R = 6371.0;
    const latRad = lat * Math.PI / 180;
    const lonRad = lon * Math.PI / 180;
    const bearingRad = bearingDeg * Math.PI / 180;
    const angularDist = distanceKm / R;

    const newLat = Math.asin(
        Math.sin(latRad) * Math.cos(angularDist) +
        Math.cos(latRad) * Math.sin(angularDist) * Math.cos(bearingRad)
    );

    const newLon = lonRad + Math.atan2(
        Math.sin(bearingRad) * Math.sin(angularDist) * Math.cos(latRad),
        Math.cos(angularDist) - Math.sin(latRad) * Math.sin(newLat)
    );

    return {
        lat: newLat * 180 / Math.PI,
        lon: newLon * 180 / Math.PI
    };
}

function highlightBeacon(beaconId) {
    Object.values(beaconMarkers).forEach(marker => {
        const el = marker.getElement();
        if (el) {
            el.style.opacity = '0.4';
        }
    });

    if (beaconMarkers[beaconId]) {
        const el = beaconMarkers[beaconId].getElement();
        if (el) {
            el.style.opacity = '1';
            el.style.transform = 'scale(1.3)';
            el.style.zIndex = '1000';
        }
    }
}

function resetBeaconHighlights() {
    Object.values(beaconMarkers).forEach(marker => {
        const el = marker.getElement();
        if (el) {
            el.style.opacity = '1';
            el.style.transform = 'scale(1)';
            el.style.zIndex = '';
        }
    });
}

async function calculateVisibility(fromId, toId) {
    try {
        const response = await fetch(
            `${API_BASE}/visibility/calculate?from_id=${fromId}&to_id=${toId}`
        );
        return await response.json();
    } catch (error) {
        console.error('Visibility calculation failed:', error);
        return null;
    }
}

function haversineDistance(lat1, lon1, lat2, lon2) {
    const R = 6371.0;
    const dLat = (lat2 - lat1) * Math.PI / 180;
    const dLon = (lon2 - lon1) * Math.PI / 180;
    const lat1Rad = lat1 * Math.PI / 180;
    const lat2Rad = lat2 * Math.PI / 180;

    const a = Math.sin(dLat / 2) * Math.sin(dLat / 2) +
              Math.cos(lat1Rad) * Math.cos(lat2Rad) *
              Math.sin(dLon / 2) * Math.sin(dLon / 2);
    const c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
    return R * c;
}

function earthCurvatureDrop(distanceKm) {
    const R = 6371000;
    const d = distanceKm * 1000;
    return (d * d) / (2 * R);
}

document.addEventListener('DOMContentLoaded', () => {
    setTimeout(setupVisibilityControls, 100);
});
