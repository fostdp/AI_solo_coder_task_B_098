let signalCanvas, signalCtx;
let signalAnimations = [];
let animationId = null;
let isSignalDemoRunning = false;

const MAX_RIPPLES = 200;
const BATCH_BUCKET_SIZE = 0.05;
const OFFSCREEN_MARGIN = 100;

let pendingAnimations = [];

function setupCanvas() {
    signalCanvas = document.getElementById('signal-canvas');
    signalCtx = signalCanvas.getContext('2d');
    resizeCanvas();
}

function resizeCanvas() {
    if (!signalCanvas) return;
    const mapContainer = document.querySelector('.map-container');
    signalCanvas.width = mapContainer.clientWidth;
    signalCanvas.height = mapContainer.clientHeight;
}

function drawSignals() {
    if (!signalCtx || !map) return;

    signalCtx.clearRect(0, 0, signalCanvas.width, signalCanvas.height);

    const now = Date.now();

    while (pendingAnimations.length > 0 && signalAnimations.length < MAX_RIPPLES) {
        signalAnimations.push(pendingAnimations.shift());
    }

    signalAnimations = signalAnimations.filter(anim => {
        const progress = (now - anim.startTime) / anim.duration;
        return progress < 1;
    });

    const buckets = new Map();

    for (let i = 0; i < signalAnimations.length; i++) {
        const anim = signalAnimations[i];
        const progress = (now - anim.startTime) / anim.duration;
        if (progress >= 1) continue;

        const center = map.latLngToContainerPoint([anim.lat, anim.lon]);
        if (center.x < -OFFSCREEN_MARGIN || center.x > signalCanvas.width + OFFSCREEN_MARGIN ||
            center.y < -OFFSCREEN_MARGIN || center.y > signalCanvas.height + OFFSCREEN_MARGIN) {
            continue;
        }

        const maxRadius = anim.maxRadius || 150;
        const currentRadius = maxRadius * progress;
        const opacity = 1 - progress;

        const bucketKey = Math.round(progress / BATCH_BUCKET_SIZE) * BATCH_BUCKET_SIZE;

        if (!buckets.has(bucketKey)) {
            buckets.set(bucketKey, []);
        }
        buckets.get(bucketKey).push({
            x: center.x,
            y: center.y,
            radius: currentRadius,
            innerRadius: currentRadius * 0.6,
            opacity: opacity,
            color: anim.color || '#e94560'
        });
    }

    for (const [bucketKey, ripples] of buckets) {
        const progress = bucketKey;
        const opacity = 1 - progress;

        signalCtx.lineWidth = 2;

        signalCtx.beginPath();
        for (let i = 0; i < ripples.length; i++) {
            const r = ripples[i];
            signalCtx.moveTo(r.x + r.radius, r.y);
            signalCtx.arc(r.x, r.y, r.radius, 0, Math.PI * 2);
        }
        signalCtx.strokeStyle = `rgba(251, 191, 36, ${0.4 * opacity})`;
        signalCtx.setLineDash([5, 3]);
        signalCtx.stroke();
        signalCtx.setLineDash([]);

        signalCtx.beginPath();
        for (let i = 0; i < ripples.length; i++) {
            const r = ripples[i];
            signalCtx.moveTo(r.x + r.innerRadius, r.y);
            signalCtx.arc(r.x, r.y, r.innerRadius, 0, Math.PI * 2);
        }
        signalCtx.strokeStyle = `rgba(74, 222, 128, ${0.6 * opacity})`;
        signalCtx.stroke();

        signalCtx.beginPath();
        for (let i = 0; i < ripples.length; i++) {
            const r = ripples[i];
            const gradient = signalCtx.createRadialGradient(
                r.x, r.y, 0,
                r.x, r.y, r.radius
            );
            gradient.addColorStop(0, `rgba(233, 69, 96, 0)`);
            gradient.addColorStop(0.3, `rgba(233, 69, 96, ${0.3 * r.opacity})`);
            gradient.addColorStop(0.6, `rgba(74, 222, 128, ${0.4 * r.opacity})`);
            gradient.addColorStop(0.85, `rgba(233, 69, 96, ${0.2 * r.opacity})`);
            gradient.addColorStop(1, `rgba(233, 69, 96, 0)`);
            signalCtx.moveTo(r.x + r.radius, r.y);
            signalCtx.arc(r.x, r.y, r.radius, 0, Math.PI * 2);
            signalCtx.fillStyle = gradient;
            signalCtx.fill();
        }
    }

    if (signalAnimations.length > 0 || pendingAnimations.length > 0) {
        if (!animationId) {
            animationId = requestAnimationFrame(animateSignalLoop);
        }
    } else {
        if (animationId) {
            cancelAnimationFrame(animationId);
            animationId = null;
        }
    }
}

function animateSignalLoop() {
    drawSignals();
    if (signalAnimations.length > 0 || pendingAnimations.length > 0) {
        animationId = requestAnimationFrame(animateSignalLoop);
    } else {
        animationId = null;
    }
}

function triggerSignal(fromBeaconId, toBeaconId) {
    const fromBeacon = beaconsData.find(b => b.id === fromBeaconId);
    if (!fromBeacon) return;

    const anim = {
        lat: fromBeacon.lat,
        lon: fromBeacon.lon,
        startTime: Date.now(),
        duration: 2000,
        maxRadius: 200,
        color: '#4ade80'
    };

    pendingAnimations.push(anim);
    if (!animationId) drawSignals();

    setTimeout(() => {
        const toBeacon = beaconsData.find(b => b.id === toBeaconId);
        if (toBeacon) {
            const relayAnim = {
                lat: toBeacon.lat,
                lon: toBeacon.lon,
                startTime: Date.now(),
                duration: 2000,
                maxRadius: 180,
                color: '#e94560'
            };
            pendingAnimations.push(relayAnim);
            if (!animationId) drawSignals();
        }
    }, 800);
}

function startSignalDemo() {
    if (isSignalDemoRunning) {
        stopSignalDemo();
        return;
    }

    isSignalDemoRunning = true;
    const btn = document.getElementById('signal-demo-btn');
    if (btn) btn.classList.add('active');

    let currentIndex = 0;
    const demoInterval = setInterval(() => {
        if (!isSignalDemoRunning || beaconsData.length === 0) {
            clearInterval(demoInterval);
            return;
        }

        const currentBeacon = beaconsData[currentIndex];
        const nextIndex = (currentIndex + 1) % beaconsData.length;

        const anim = {
            lat: currentBeacon.lat,
            lon: currentBeacon.lon,
            startTime: Date.now(),
            duration: 2500,
            maxRadius: 220,
            color: '#fbbf24'
        };

        pendingAnimations.push(anim);
        if (!animationId) drawSignals();

        currentIndex = nextIndex;
    }, 1000);

    window._signalDemoInterval = demoInterval;
}

function stopSignalDemo() {
    isSignalDemoRunning = false;
    const btn = document.getElementById('signal-demo-btn');
    if (btn) btn.classList.remove('active');

    if (window._signalDemoInterval) {
        clearInterval(window._signalDemoInterval);
        window._signalDemoInterval = null;
    }
}

function drawLinkSignal(fromId, toId) {
    const fromBeacon = beaconsData.find(b => b.id === fromId);
    const toBeacon = beaconsData.find(b => b.id === toId);

    if (!fromBeacon || !toBeacon || !map || !signalCtx) return;

    const fromPoint = map.latLngToContainerPoint([fromBeacon.lat, fromBeacon.lon]);
    const toPoint = map.latLngToContainerPoint([toBeacon.lat, toBeacon.lon]);

    let progress = 0;
    const duration = 1500;
    const startTime = Date.now();

    function animatePulse() {
        const elapsed = Date.now() - startTime;
        progress = elapsed / duration;

        if (progress >= 1) return;

        const currentX = fromPoint.x + (toPoint.x - fromPoint.x) * progress;
        const currentY = fromPoint.y + (toPoint.y - fromPoint.y) * progress;

        signalCtx.save();
        signalCtx.beginPath();
        signalCtx.arc(currentX, currentY, 8, 0, Math.PI * 2);
        signalCtx.fillStyle = `rgba(74, 222, 128, ${0.8 * (1 - progress * 0.5)})`;
        signalCtx.fill();

        signalCtx.beginPath();
        signalCtx.arc(currentX, currentY, 15, 0, Math.PI * 2);
        signalCtx.strokeStyle = `rgba(251, 191, 36, ${0.6 * (1 - progress)})`;
        signalCtx.lineWidth = 2;
        signalCtx.stroke();
        signalCtx.restore();

        requestAnimationFrame(animatePulse);
    }

    animatePulse();
}

function broadcastSignal(beaconId) {
    const beacon = beaconsData.find(b => b.id === beaconId);
    if (!beacon) return;

    for (let i = 0; i < 5; i++) {
        setTimeout(() => {
            const anim = {
                lat: beacon.lat,
                lon: beacon.lon,
                startTime: Date.now(),
                duration: 1800,
                maxRadius: 100 + i * 40,
                color: '#4ade80'
            };
            pendingAnimations.push(anim);
        }, i * 200);
    }

    if (!animationId) setTimeout(() => drawSignals(), 50);
}

window.triggerSignal = triggerSignal;
window.startSignalDemo = startSignalDemo;
window.stopSignalDemo = stopSignalDemo;
window.broadcastSignal = broadcastSignal;
window.drawLinkSignal = drawLinkSignal;
