// O-RAN Intent-MANO Frontend Application

const API_BASE = window.location.origin;
let currentDeployment = null;

// 初始化
document.addEventListener('DOMContentLoaded', () => {
    console.log('O-RAN Intent-MANO 初始化...');
    checkSystemHealth();
    loadActiveSlices();
    updateMetrics();

    // 定期更新
    setInterval(checkSystemHealth, 30000);
    setInterval(loadActiveSlices, 10000);
    setInterval(updateMetrics, 15000);
});

// 系統健康檢查
async function checkSystemHealth() {
    try {
        const response = await fetch(`${API_BASE}/health`);
        const data = await response.json();

        const statusDot = document.getElementById('systemStatus');
        const statusText = document.getElementById('statusText');
        const versionEl = document.getElementById('version');

        if (data.status === 'healthy') {
            statusDot.classList.remove('error');
            statusText.textContent = '系統正常';
            if (data.version) {
                versionEl.textContent = data.version;
            }
        } else {
            statusDot.classList.add('error');
            statusText.textContent = '系統異常';
        }
    } catch (error) {
        console.error('健康檢查失敗:', error);
        document.getElementById('systemStatus').classList.add('error');
        document.getElementById('statusText').textContent = '連線失敗';
    }
}

// 載入活動切片
async function loadActiveSlices() {
    try {
        const response = await fetch(`${API_BASE}/api/v1/slices`);
        const data = await response.json();

        const container = document.getElementById('slicesContainer');

        if (!data.slices || data.slices.length === 0) {
            container.innerHTML = '<div class="loading">目前沒有活動切片</div>';
            return;
        }

        container.innerHTML = data.slices.map(slice => `
            <div class="slice-card ${slice.slice_type}">
                <div class="slice-header">
                    <span class="slice-id">${slice.slice_id}</span>
                    <span class="slice-type">${slice.slice_type.toUpperCase()}</span>
                </div>
                <div class="slice-specs">
                    <div class="spec-item">
                        <span class="spec-label">頻寬</span>
                        <div class="spec-value">${slice.qos.bandwidth} Mbps</div>
                    </div>
                    <div class="spec-item">
                        <span class="spec-label">延遲</span>
                        <div class="spec-value">${slice.qos.latency} ms</div>
                    </div>
                    <div class="spec-item">
                        <span class="spec-label">狀態</span>
                        <div class="spec-value">${translateStatus(slice.status)}</div>
                    </div>
                    ${slice.deployment_time ? `
                    <div class="spec-item">
                        <span class="spec-label">部署時間</span>
                        <div class="spec-value">${formatTime(slice.deployment_time)}</div>
                    </div>
                    ` : ''}
                </div>
            </div>
        `).join('');

    } catch (error) {
        console.error('載入切片失敗:', error);
        document.getElementById('slicesContainer').innerHTML =
            '<div class="loading">載入失敗，請稍後重試</div>';
    }
}

// 更新系統指標
async function updateMetrics() {
    try {
        const response = await fetch(`${API_BASE}/api/v1/slices`);
        const data = await response.json();

        if (data.slices) {
            document.getElementById('totalSlices').textContent = data.total || 0;
            document.getElementById('activeSlices').textContent =
                data.slices.filter(s => s.status === 'active').length;

            // 計算平均延遲
            const avgLat = data.slices.reduce((sum, s) => sum + (s.qos.latency || 0), 0) / data.slices.length;
            document.getElementById('avgLatency').textContent = avgLat.toFixed(1) + ' ms';

            // 計算總頻寬
            const totalBW = data.slices.reduce((sum, s) => sum + (s.qos.bandwidth || 0), 0);
            document.getElementById('totalBandwidth').textContent = totalBW.toFixed(0) + ' Mbps';
        }
    } catch (error) {
        console.error('更新指標失敗:', error);
    }
}

// 載入預設場景
function loadScenario(type) {
    const scenarios = {
        'embb': '我需要支援 4K 高畫質影音串流，延遲不超過 20ms，頻寬至少 100Mbps',
        'urllc': '自動駕駛車輛需要超低延遲 1ms 的網路連接，可靠度要達到 99.999%，頻寬 10Mbps',
        'mmtc': '智慧城市 IoT 感測器網路，需要連接 10000 個設備，每個設備 1Mbps，延遲 100ms 可接受'
    };

    const textarea = document.getElementById('naturalLanguageInput');
    textarea.value = scenarios[type] || '';
    textarea.focus();

    // 立即顯示解析預覽
    showParsingPreview(scenarios[type]);
}

// 聚焦自然語言輸入
function focusNaturalInput() {
    document.getElementById('naturalLanguageInput').focus();
}

// 設定範例
function setExample(text) {
    document.getElementById('naturalLanguageInput').value = text;
    showParsingPreview(text);
}

// 清除輸入
function clearInput() {
    document.getElementById('naturalLanguageInput').value = '';
    document.getElementById('pipeline').style.display = 'none';
}

// 顯示解析預覽
function showParsingPreview(text) {
    const result = nlpParser.parse(text);

    if (result.success) {
        showToast(`Intent 解析完成 (信心度: ${result.confidence}%)`, 'success');
        console.log('解析結果:', result.intent);
    }
}

// 解析並部署
async function parseAndDeploy() {
    const text = document.getElementById('naturalLanguageInput').value;

    if (!text.trim()) {
        showToast('請輸入自然語言描述', 'error');
        return;
    }

    const deployBtn = document.getElementById('deployBtn');
    deployBtn.disabled = true;
    deployBtn.innerHTML = '<i class="fas fa-spinner fa-spin"></i> 處理中...';

    // 顯示處理流程
    document.getElementById('pipeline').style.display = 'block';

    try {
        // 步驟 1: 解析自然語言
        await updatePipelineStep(1, 'processing', '正在解析自然語言...', null);
        const parseResult = nlpParser.parse(text);

        if (!parseResult.success) {
            throw new Error(parseResult.error);
        }

        const explanation = nlpParser.explainIntent(parseResult.intent);
        await updatePipelineStep(1, 'success', '解析完成', `
            <pre>${explanation}</pre>
            <div style="margin-top: 0.5rem; color: #6b7280;">信心度: ${parseResult.confidence}%</div>
        `);

        await sleep(800);

        // 步驟 2: 生成 QoS Profile
        await updatePipelineStep(2, 'processing', '正在生成 QoS Profile...', null);
        await sleep(500);

        const qosProfile = {
            bandwidth: parseResult.intent.bandwidth,
            latency: parseResult.intent.latency,
            slice_type: parseResult.intent.slice_type
        };

        if (parseResult.intent.reliability) {
            qosProfile.reliability = parseResult.intent.reliability;
        }
        if (parseResult.intent.jitter) {
            qosProfile.jitter = parseResult.intent.jitter;
        }
        if (parseResult.intent.packet_loss) {
            qosProfile.packet_loss = parseResult.intent.packet_loss;
        }

        await updatePipelineStep(2, 'success', 'QoS Profile 生成完成', `
            <pre>${JSON.stringify(qosProfile, null, 2)}</pre>
        `);

        await sleep(800);

        // 步驟 3: 智能資源配置
        await updatePipelineStep(3, 'processing', '正在計算最佳資源配置...', null);
        await sleep(800);

        const placement = {
            site: qosProfile.latency <= 5 ? 'Edge Site 01' : 'Regional Site 01',
            resources: {
                cpu_cores: Math.ceil(qosProfile.bandwidth * 0.1),
                memory_gb: Math.ceil(qosProfile.bandwidth * 0.1),
                storage_gb: Math.ceil(qosProfile.bandwidth * 0.5)
            }
        };

        await updatePipelineStep(3, 'success', '資源配置完成', `
            <div><strong>部署位置:</strong> ${placement.site}</div>
            <div style="margin-top: 0.5rem;"><strong>資源分配:</strong></div>
            <pre>${JSON.stringify(placement.resources, null, 2)}</pre>
        `);

        await sleep(800);

        // 步驟 4: 部署切片
        await updatePipelineStep(4, 'processing', '正在部署網路切片...', null);

        const deployResponse = await fetch(`${API_BASE}/api/v1/intents`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(qosProfile)
        });

        if (!deployResponse.ok) {
            throw new Error(`部署失敗: ${deployResponse.statusText}`);
        }

        const deployResult = await deployResponse.json();
        currentDeployment = deployResult;

        await updatePipelineStep(4, 'success', '切片部署完成', `
            <div><strong>Slice ID:</strong> ${deployResult.slice_id}</div>
            <div><strong>狀態:</strong> ${translateStatus(deployResult.status)}</div>
            <div style="margin-top: 0.5rem; color: #6b7280;">時間戳: ${new Date(deployResult.timestamp * 1000).toLocaleString('zh-TW')}</div>
        `);

        await sleep(500);

        // 步驟 5: 完成
        await updatePipelineStep(5, 'success', '部署流程完成！', `
            <div style="color: #10b981; font-weight: 600;">
                <i class="fas fa-check-circle"></i> 網路切片已成功部署並運行
            </div>
            <div style="margin-top: 0.5rem;">
                <strong>Slice ID:</strong> ${deployResult.slice_id}<br>
                <strong>類型:</strong> ${qosProfile.slice_type.toUpperCase()}<br>
                <strong>頻寬:</strong> ${qosProfile.bandwidth} Mbps<br>
                <strong>延遲:</strong> ${qosProfile.latency} ms
            </div>
        `);

        showToast('切片部署成功！', 'success');

        // 重新載入活動切片列表
        setTimeout(() => {
            loadActiveSlices();
            updateMetrics();
        }, 1000);

    } catch (error) {
        console.error('部署失敗:', error);
        showToast(`部署失敗: ${error.message}`, 'error');

        // 顯示錯誤在當前步驟
        const currentStep = document.querySelector('.pipeline-step.processing');
        if (currentStep) {
            const stepNum = currentStep.id.replace('step', '');
            await updatePipelineStep(parseInt(stepNum), 'error', '處理失敗', `
                <div style="color: #ef4444;">
                    <i class="fas fa-exclamation-circle"></i> ${error.message}
                </div>
            `);
        }
    } finally {
        deployBtn.disabled = false;
        deployBtn.innerHTML = '<i class="fas fa-paper-plane"></i> 解析並部署';
    }
}

// 更新流程步驟
async function updatePipelineStep(stepNum, status, statusText, details) {
    const step = document.getElementById(`step${stepNum}`);
    const stepStatus = step.querySelector('.step-status');
    const stepDetails = step.querySelector('.step-details');

    // 移除所有狀態類別
    step.classList.remove('processing', 'success', 'error');

    // 添加新狀態
    if (status) {
        step.classList.add(status);
    }

    // 更新狀態文字
    if (statusText) {
        stepStatus.textContent = statusText;
    }

    // 更新詳細資訊
    if (details) {
        stepDetails.innerHTML = details;
        stepDetails.style.display = 'block';
    }

    // 平滑滾動到當前步驟
    step.scrollIntoView({ behavior: 'smooth', block: 'center' });

    return new Promise(resolve => setTimeout(resolve, 100));
}

// 顯示通知
function showToast(message, type = 'info') {
    const container = document.getElementById('toastContainer');
    const toast = document.createElement('div');
    toast.className = `toast ${type}`;

    const icons = {
        success: 'fa-check-circle',
        error: 'fa-exclamation-circle',
        warning: 'fa-exclamation-triangle',
        info: 'fa-info-circle'
    };

    toast.innerHTML = `
        <i class="fas ${icons[type]}"></i>
        <span>${message}</span>
    `;

    container.appendChild(toast);

    // 3 秒後自動移除
    setTimeout(() => {
        toast.style.animation = 'slideIn 0.3s ease-out reverse';
        setTimeout(() => toast.remove(), 300);
    }, 3000);
}

// 輔助函數
function translateStatus(status) {
    const statusMap = {
        'active': '運行中',
        'created': '已創建',
        'deploying': '部署中',
        'failed': '失敗',
        'pending': '等待中'
    };
    return statusMap[status] || status;
}

function formatTime(timestamp) {
    const date = new Date(timestamp * 1000);
    const now = new Date();
    const diff = Math.floor((now - date) / 1000);

    if (diff < 60) return `${diff} 秒前`;
    if (diff < 3600) return `${Math.floor(diff / 60)} 分鐘前`;
    if (diff < 86400) return `${Math.floor(diff / 3600)} 小時前`;
    return date.toLocaleDateString('zh-TW');
}

function sleep(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
}