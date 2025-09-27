// Natural Language Intent Parser
// 使用規則匹配從自然語言中提取 QoS 參數

class NLPParser {
    constructor() {
        // 關鍵字模式
        this.patterns = {
            // 場景類型
            sliceTypes: {
                'embb': /4K|8K|影音|視訊|串流|高畫質|超高畫質|視頻|直播/i,
                'urllc': /自動駕駛|工業|遠程手術|即時|實時|低延遲|超低延遲|可靠|機器人/i,
                'mmtc': /IoT|物聯網|感測器|智慧城市|智慧農業|大規模|海量|設備/i
            },

            // 數值提取
            bandwidth: /(\d+(?:\.\d+)?)\s*(Mbps|mbps|M|MB\/s|Gb|Gbps)/i,
            latency: /延遲|latency.*?(\d+(?:\.\d+)?)\s*(ms|毫秒)|(\d+(?:\.\d+)?)\s*(ms|毫秒).*?延遲/i,
            reliability: /可靠度|可靠性|reliability.*?(\d+(?:\.\d+)?)\s*%|(\d+(?:\.\d+)?)\s*個.*?9|99\.9+/i,
            jitter: /抖動|jitter.*?(\d+(?:\.\d+)?)\s*(ms|毫秒)/i,
            packetLoss: /丟包率|封包遺失|packet.*?loss.*?(\d+(?:\.\d+)?)\s*%/i,
            deviceCount: /(\d+)\s*(?:個|台)?.*?設備|設備.*?(\d+)\s*(?:個|台)?/i,

            // 品質關鍵字
            quality: {
                high: /高|超高|極高|4K|8K|HD|UHD/i,
                medium: /中|普通|一般|標準/i,
                low: /低|基本|簡單/i
            }
        };

        // 預設值
        this.defaults = {
            embb: {
                bandwidth: 100,
                latency: 20,
                jitter: 5,
                packet_loss: 0.001
            },
            urllc: {
                bandwidth: 10,
                latency: 1,
                reliability: 0.99999,
                jitter: 0.5,
                packet_loss: 0.0001
            },
            mmtc: {
                bandwidth: 1,
                latency: 100,
                jitter: 10,
                packet_loss: 0.01
            }
        };
    }

    parse(text) {
        if (!text || text.trim() === '') {
            return {
                error: '請輸入自然語言描述',
                intent: null
            };
        }

        const intent = {
            slice_type: this.detectSliceType(text),
            bandwidth: this.extractBandwidth(text),
            latency: this.extractLatency(text),
            reliability: this.extractReliability(text),
            jitter: this.extractJitter(text),
            packet_loss: this.extractPacketLoss(text)
        };

        // 使用預設值填充缺失的參數
        const defaults = this.defaults[intent.slice_type] || this.defaults.embb;

        if (!intent.bandwidth) intent.bandwidth = defaults.bandwidth;
        if (!intent.latency) intent.latency = defaults.latency;
        if (!intent.reliability && defaults.reliability) {
            intent.reliability = defaults.reliability;
        }
        if (!intent.jitter && defaults.jitter) {
            intent.jitter = defaults.jitter;
        }
        if (!intent.packet_loss && defaults.packet_loss) {
            intent.packet_loss = defaults.packet_loss;
        }

        return {
            success: true,
            intent: intent,
            confidence: this.calculateConfidence(text, intent)
        };
    }

    detectSliceType(text) {
        for (const [type, pattern] of Object.entries(this.patterns.sliceTypes)) {
            if (pattern.test(text)) {
                return type;
            }
        }
        return 'embb'; // 預設
    }

    extractBandwidth(text) {
        const match = text.match(this.patterns.bandwidth);
        if (match) {
            let value = parseFloat(match[1]);
            const unit = match[2].toUpperCase();

            // 轉換為 Mbps
            if (unit.includes('G')) {
                value *= 1000;
            }

            return value;
        }
        return null;
    }

    extractLatency(text) {
        const match = text.match(this.patterns.latency);
        if (match) {
            // 找到第一個匹配的數字
            const value = match[1] || match[3];
            return parseFloat(value);
        }
        return null;
    }

    extractReliability(text) {
        const match = text.match(this.patterns.reliability);
        if (match) {
            if (match[0].includes('99.9')) {
                // 計算 9 的數量
                const nines = (match[0].match(/9/g) || []).length;
                return parseFloat('0.' + '9'.repeat(nines));
            } else if (match[1]) {
                let value = parseFloat(match[1]);
                // 如果是百分比，轉換為小數
                if (value > 1) {
                    value = value / 100;
                }
                return value;
            }
        }
        return null;
    }

    extractJitter(text) {
        const match = text.match(this.patterns.jitter);
        if (match) {
            return parseFloat(match[1]);
        }
        return null;
    }

    extractPacketLoss(text) {
        const match = text.match(this.patterns.packetLoss);
        if (match) {
            let value = parseFloat(match[1]);
            // 轉換為小數
            if (value > 0.1) {
                value = value / 100;
            }
            return value;
        }
        return null;
    }

    calculateConfidence(text, intent) {
        let score = 0;
        let total = 0;

        // 檢查每個參數
        const checks = [
            { key: 'slice_type', weight: 2 },
            { key: 'bandwidth', weight: 2 },
            { key: 'latency', weight: 2 },
            { key: 'reliability', weight: 1 },
            { key: 'jitter', weight: 1 },
            { key: 'packet_loss', weight: 1 }
        ];

        checks.forEach(check => {
            total += check.weight;
            if (intent[check.key]) {
                // 檢查是否是從文本中提取的（不是預設值）
                if (check.key === 'slice_type') {
                    const pattern = this.patterns.sliceTypes[intent.slice_type];
                    if (pattern && pattern.test(text)) {
                        score += check.weight;
                    }
                } else {
                    const defaults = this.defaults[intent.slice_type];
                    if (!defaults || intent[check.key] !== defaults[check.key]) {
                        score += check.weight;
                    }
                }
            }
        });

        return Math.round((score / total) * 100);
    }

    // 生成人類可讀的解釋
    explainIntent(intent) {
        const sliceNames = {
            'embb': 'eMBB (增強型行動寬頻)',
            'urllc': 'URLLC (超可靠低延遲通訊)',
            'mmtc': 'mMTC (大規模機器通訊)'
        };

        let explanation = `檢測到場景類型: ${sliceNames[intent.slice_type]}\n\n`;
        explanation += `QoS 參數:\n`;
        explanation += `- 頻寬: ${intent.bandwidth} Mbps\n`;
        explanation += `- 延遲: ${intent.latency} ms\n`;

        if (intent.reliability) {
            const nines = intent.reliability.toString().split('.')[1]?.match(/9/g)?.length || 0;
            explanation += `- 可靠度: ${nines}個9 (${(intent.reliability * 100).toFixed(3)}%)\n`;
        }

        if (intent.jitter) {
            explanation += `- 抖動: ${intent.jitter} ms\n`;
        }

        if (intent.packet_loss) {
            explanation += `- 丟包率: ${(intent.packet_loss * 100).toFixed(3)}%\n`;
        }

        return explanation;
    }
}

// 全局實例
const nlpParser = new NLPParser();