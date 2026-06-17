let signalCanvas, signalCtx;
let signalAnimations = [];
let animationId = null;
let isSignalDemoRunning = false;

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

    signalAnimations = signalAnimations.filter(anim => {
        const progress = (Date.now() - anim.startTime) / anim.duration;
        if (progress >= 1) return false;

        drawSignalRipple(anim, progress);
        return true;
    });

    if (signalAnimations.length > 0) {
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
    if (signalAnimations.length > 0) {
        animationId = requestAnimationFrame(animateSignalLoop);
    }
}

function drawSignalRipple(anim, progress) {
    const center = map.latLngToContainerPoint([anim.lat, anim.lon]);
    const maxRadius = anim.maxRadius || 150;
    const currentRadius = maxRadius * progress;
    const opacity = 1 - progress;

    signalCtx.save();

    const gradient = signalCtx.createRadialGradient(
        center.x, center.y, 0,
        center.x, center.y, currentRadius
    );

    const color = anim.color || '#e94560';
    gradient.addColorStop(0, `rgba(233, 69, 96, 0)`);
    gradient.addColorStop(0.3, `rgba(233, 69, 96, ${0.3 * opacity})`);
    gradient.addColorStop(0.6, `rgba(74, 222, 128, ${0.4 * opacity})`);
    gradient.addColorStop(0.85, `rgba(233, 69, 96, ${0.2 * opacity})`);
    gradient.addColorStop(1, `rgba(233, 69, 96, 0)`);

    signalCtx.beginPath();
    signalCtx.arc(center.x, center.y, currentRadius, 0, Math.PI * 2);
    signalCtx.fillStyle = gradient;
    signalCtx.fill();

    signalCtx.beginPath();
    signalCtx.arc(center.x, center.y, currentRadius * 0.6, 0, Math.PI * 2);
    signalCtx.strokeStyle = `rgba(74, 222, 128, ${0.6 * opacity})`;
    signalCtx.lineWidth = 2;
    signalCtx.stroke();

    signalCtx.beginPath();
    signalCtx.arc(center.x, center.y, currentRadius, 0, Math.PI * 2);
    signalCtx.strokeStyle = `rgba(251, 191, 36, ${0.4 * opacity})`;
    signalCtx.lineWidth = 1.5;
    signalCtx.setLineDash([5, 3]);
    signalCtx.stroke();
    signalCtx.setLineDash([]);

    signalCtx.restore();
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

    signalAnimations.push(anim);
    drawSignals();

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
            signalAnimations.push(relayAnim);
            drawSignals();
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

        signalAnimations.push(anim);
        drawSignals();

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
            signalAnimations.push(anim);
        }, i * 200);
    }

    setTimeout(() => drawSignals(), 50);
}

window.triggerSignal = triggerSignal;
window.startSignalDemo = startSignalDemo;
window.stopSignalDemo = stopSignalDemo;
window.broadcastSignal = broadcastSignal;
window.drawLinkSignal = drawLinkSignal;
