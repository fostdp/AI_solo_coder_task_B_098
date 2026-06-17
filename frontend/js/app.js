document.addEventListener('DOMContentLoaded', () => {
    setTimeout(initApp, 200);
});

function initApp() {
    setupMapControls();
    setupAnalysisControls();
    setupRefreshButton();
}

function setupMapControls() {
    const toggleLinksBtn = document.getElementById('toggle-links-btn');
    const toggleLabelsBtn = document.getElementById('toggle-labels-btn');
    const signalDemoBtn = document.getElementById('signal-demo-btn');

    if (toggleLinksBtn) {
        toggleLinksBtn.addEventListener('click', () => {
            if (typeof toggleLinks === 'function') {
                toggleLinks();
            }
        });
    }

    if (toggleLabelsBtn) {
        toggleLabelsBtn.addEventListener('click', () => {
            if (typeof toggleLabels === 'function') {
                toggleLabels();
            }
        });
    }

    if (signalDemoBtn) {
        signalDemoBtn.addEventListener('click', () => {
            if (typeof startSignalDemo === 'function' && typeof stopSignalDemo === 'function') {
                const isRunning = signalDemoBtn.classList.contains('active');
                if (isRunning) {
                    stopSignalDemo();
                } else {
                    startSignalDemo();
                }
            }
        });
    }
}

function setupAnalysisControls() {
    const analyzeBtn = document.getElementById('analyze-btn');
    const iterationsSlider = document.getElementById('iterations-slider');
    const iterationsValue = document.getElementById('iterations-value');

    if (iterationsSlider && iterationsValue) {
        iterationsSlider.addEventListener('input', () => {
            iterationsValue.textContent = iterationsSlider.value;
        });
    }

    if (analyzeBtn) {
        analyzeBtn.addEventListener('click', runReliabilityAnalysis);
    }
}

async function runReliabilityAnalysis() {
    const weatherSelect = document.getElementById('weather-select');
    const iterationsSlider = document.getElementById('iterations-slider');
    const analyzeBtn = document.getElementById('analyze-btn');

    if (!weatherSelect || !iterationsSlider) return;

    const weatherFactor = parseFloat(weatherSelect.value);
    const iterations = parseInt(iterationsSlider.value);

    analyzeBtn.disabled = true;
    analyzeBtn.textContent = '分析中...';

    try {
        const response = await fetch(
            `${API_BASE}/network/reliability?iterations=${iterations}&weather_factor=${weatherFactor}`
        );
        const data = await response.json();

        displayAnalysisResults(data);
        updateLinkColors(data);

        if (data.monte_carlo && data.monte_carlo.success_rate < 0.7) {
            showAlertToast('网络可靠性较低，建议检查关键链路');
        }

    } catch (error) {
        console.error('Analysis failed:', error);
        showAlertToast('可靠性分析失败');
    } finally {
        analyzeBtn.disabled = false;
        analyzeBtn.textContent = '运行可靠性分析';
    }
}

function displayAnalysisResults(data) {
    if (data.monte_carlo) {
        const reliability = (data.monte_carlo.success_rate * 100).toFixed(1);
        document.getElementById('reliability-value').textContent = reliability + '%';

        const reliabilityEl = document.getElementById('reliability-value');
        if (data.monte_carlo.success_rate >= 0.8) {
            reliabilityEl.style.color = '#4ade80';
        } else if (data.monte_carlo.success_rate >= 0.6) {
            reliabilityEl.style.color = '#fbbf24';
        } else {
            reliabilityEl.style.color = '#ef4444';
        }
    }

    if (data.network_metrics) {
        const metrics = data.network_metrics;
        const linkInfo = document.getElementById('link-info');
        
        let html = '<h4 style="color: #e94560; margin-bottom: 10px;">📊 网络分析结果</h4>';
        
        if (data.monte_carlo) {
            html += `
                <div class="info-row">
                    <span class="info-label">蒙特卡洛成功率</span>
                    <span class="info-value">${(data.monte_carlo.success_rate * 100).toFixed(2)}%</span>
                </div>
            `;
            
            if (data.monte_carlo.confidence_interval) {
                const ci = data.monte_carlo.confidence_interval;
                html += `
                    <div class="info-row">
                        <span class="info-label">95%置信区间</span>
                        <span class="info-value">${(ci[0]*100).toFixed(1)}% - ${(ci[1]*100).toFixed(1)}%</span>
                    </div>
                `;
            }
        }
        
        html += `
            <div class="info-row">
                <span class="info-label">节点数量</span>
                <span class="info-value">${metrics.node_count || '--'}</span>
            </div>
            <div class="info-row">
                <span class="info-label">链路数量</span>
                <span class="info-value">${metrics.link_count || '--'}</span>
            </div>
            <div class="info-row">
                <span class="info-label">连通度指数</span>
                <span class="info-value" style="color: ${metrics.connectivity_index >= 0.8 ? '#4ade80' : '#fbbf24'}">
                    ${(metrics.connectivity_index * 100).toFixed(1)}%
                </span>
            </div>
            <div class="info-row">
                <span class="info-label">平均路径长度</span>
                <span class="info-value">${metrics.avg_path_length?.toFixed(2) || '--'}</span>
            </div>
            <div class="info-row">
                <span class="info-label">关键链路数</span>
                <span class="info-value" style="color: #e94560;">${metrics.critical_links || 0}</span>
            </div>
            <div class="info-row">
                <span class="info-label">平均链路可靠性</span>
                <span class="info-value">${(metrics.avg_link_reliability * 100).toFixed(1)}%</span>
            </div>
        `;
        
        linkInfo.innerHTML = html;
    }
}

function updateLinkColors(data) {
    if (!data || !data.monte_carlo) return;
}

function setupRefreshButton() {
    const refreshBtn = document.getElementById('refresh-btn');
    if (refreshBtn) {
        refreshBtn.addEventListener('click', refreshAllData);
    }
}

async function refreshAllData() {
    const refreshBtn = document.getElementById('refresh-btn');
    if (refreshBtn) {
        refreshBtn.style.animation = 'spin 1s linear';
        setTimeout(() => {
            refreshBtn.style.animation = '';
        }, 1000);
    }

    if (typeof updateStats === 'function') {
        updateStats();
    }

    if (typeof loadNetworkTopology === 'function') {
        loadNetworkTopology();
    }
}

function showAlertToast(message) {
    const toast = document.createElement('div');
    toast.style.cssText = `
        position: fixed;
        top: 80px;
        right: 20px;
        background: rgba(239, 68, 68, 0.95);
        color: white;
        padding: 12px 20px;
        border-radius: 8px;
        z-index: 10000;
        font-size: 14px;
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
        transform: translateX(120%);
        transition: transform 0.3s ease;
    `;
    toast.textContent = '⚠️ ' + message;
    
    document.body.appendChild(toast);
    
    setTimeout(() => {
        toast.style.transform = 'translateX(0)';
    }, 10);
    
    setTimeout(() => {
        toast.style.transform = 'translateX(120%)';
        setTimeout(() => toast.remove(), 300);
    }, 4000);
}

const style = document.createElement('style');
style.textContent = `
    @keyframes spin {
        from { transform: rotate(0deg); }
        to { transform: rotate(360deg); }
    }
    
    .custom-beacon-icon {
        background: transparent;
        border: none;
    }
    
    .beacon-marker {
        position: relative;
        width: 20px;
        height: 20px;
        transition: transform 0.3s ease;
    }
    
    .beacon-marker:hover {
        transform: scale(1.2);
    }
    
    .beacon-core {
        position: absolute;
        top: 50%;
        left: 50%;
        transform: translate(-50%, -50%);
        width: 12px;
        height: 12px;
        background: #4ade80;
        border-radius: 50%;
        box-shadow: 0 0 10px #4ade80, 0 0 20px rgba(74, 222, 128, 0.5);
        z-index: 2;
    }
    
    .beacon-marker.critical .beacon-core {
        background: #e94560;
        box-shadow: 0 0 10px #e94560, 0 0 20px rgba(233, 69, 96, 0.5);
        width: 14px;
        height: 14px;
    }
    
    .beacon-pulse {
        position: absolute;
        top: 50%;
        left: 50%;
        transform: translate(-50%, -50%);
        width: 20px;
        height: 20px;
        border-radius: 50%;
        background: rgba(74, 222, 128, 0.3);
        animation: beacon-pulse 2s infinite;
    }
    
    .beacon-marker.critical .beacon-pulse {
        background: rgba(233, 69, 96, 0.3);
    }
    
    @keyframes beacon-pulse {
        0% {
            transform: translate(-50%, -50%) scale(1);
            opacity: 1;
        }
        100% {
            transform: translate(-50%, -50%) scale(2.5);
            opacity: 0;
        }
    
    .beacon-label {
        position: absolute;
        top: 100%;
        left: 50%;
        transform: translateX(-50%);
        white-space: nowrap;
        font-size: 10px;
        color: #e2e8f0;
        background: rgba(22, 33, 62, 0.9);
        padding: 2px 6px;
        border-radius: 3px;
        margin-top: 2px;
        pointer-events: none;
        text-shadow: 0 1px 2px rgba(0, 0, 0, 0.8);
    }
    
    .leaflet-popup-content-wrapper {
        background: #16213e !important;
        color: #eee !important;
        border-radius: 8px !important;
        border: 1px solid #e94560 !important;
    }
    
    .leaflet-popup-tip {
        background: #16213e !important;
    }
    
    .leaflet-popup-content h4 {
        color: #e94560;
        margin-bottom: 8px;
    }
    
    .leaflet-popup-content .info-label {
        color: #94a3b8;
        display: inline-block;
        width: 60px;
    }
`;

document.head.appendChild(style);
