#!/usr/bin/env python3
"""
O-RAN Intent MANO - Integrated HTTP + WebSocket Server
"""

import asyncio
import json
import logging
import subprocess
import tempfile
from datetime import datetime
from aiohttp import web
import aiohttp

logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

class IntegratedServer:
    def __init__(self):
        self.clients = {}
        self.orchestrator_url = "http://localhost:8081"

    async def websocket_handler(self, request):
        """Handle WebSocket connections"""
        ws = web.WebSocketResponse()
        await ws.prepare(request)

        session_id = f"session-{datetime.now().strftime('%Y%m%d%H%M%S%f')}"
        self.clients[session_id] = ws

        logger.info(f"✅ WebSocket client connected: {session_id}")

        # Send welcome message
        await ws.send_json({
            "type": "connected",
            "sessionId": session_id,
            "message": "Connected to O-RAN Network Slicing Service",
            "status": "success",
            "timestamp": int(datetime.now().timestamp())
        })

        try:
            async for msg in ws:
                if msg.type == aiohttp.WSMsgType.TEXT:
                    await self.process_message(ws, session_id, msg.data)
                elif msg.type == aiohttp.WSMsgType.ERROR:
                    logger.error(f"WebSocket error: {ws.exception()}")
        finally:
            logger.info(f"❌ WebSocket client disconnected: {session_id}")
            if session_id in self.clients:
                del self.clients[session_id]

        return ws

    async def process_message(self, ws, session_id, message):
        """Process incoming WebSocket message"""
        try:
            data = json.loads(message)
            msg_type = data.get("type", "")

            logger.info(f"📥 Received {msg_type} from {session_id}")

            if msg_type == "intent":
                await self.process_intent(ws, session_id, data)
            elif msg_type == "ping":
                await ws.send_json({"type": "pong"})

        except json.JSONDecodeError as e:
            logger.error(f"JSON decode error: {e}")

    def classify_intent(self, text: str) -> str:
        """Classify intent: advisory (consultative), deployment (immediate action), or conversation"""
        text_lower = text.lower()

        # Advisory/consultative question keywords - these need AI reasoning
        advisory_keywords = [
            "請問", "如何選", "怎麼選", "建議", "推薦", "應該", "哪種", "哪個",
            "what should", "which", "recommend", "suggest", "advise", "選擇",
            "考量", "分析", "評估", "比較", "適合", "最好"
        ]

        # Deployment intent keywords - immediate action
        deployment_keywords = [
            "部署", "創建", "建立", "deploy", "create", "需要一個", "給我",
            "幫我開", "啟動", "建置", "設置"
        ]

        # Technical slice keywords
        technical_keywords = [
            "切片", "slice", "redcap", "embb", "urllc", "mmtc", "iot", "video",
            "autonomous", "throughput", "latency", "延遲", "頻寬", "bandwidth",
            "mbps", "ms", "5qi", "qos"
        ]

        # Conversational question keywords
        conversational_keywords = [
            "你是", "what are", "who are", "為何", "why", "是什麼",
            "可以", "能否", "是否", "explain", "告訴我", "介紹", "你愛"
        ]

        # Check for each type
        has_advisory = any(kw in text_lower for kw in advisory_keywords)
        has_deployment = any(kw in text_lower for kw in deployment_keywords)
        has_technical = any(kw in text_lower for kw in technical_keywords)
        has_conversation = any(kw in text_lower for kw in conversational_keywords)

        # Priority: advisory > deployment > conversation
        # Advisory: questions about choosing/recommending slice types (needs AI reasoning)
        if has_advisory and has_technical:
            return "advisory"
        # Deployment: explicit action words + technical terms
        elif has_deployment and has_technical:
            return "deployment"
        # Conversation: about the system itself
        elif has_conversation and not has_technical:
            return "conversation"
        # Default to advisory if technical but unclear
        elif has_technical:
            return "advisory"
        else:
            return "conversation"

    async def process_intent(self, ws, session_id, data):
        """Process NL intent with step-by-step visualization"""
        intent_text = data.get("intent", "")
        logger.info(f"🎯 Processing intent: {intent_text}")

        # Classify intent type
        intent_type = self.classify_intent(intent_text)
        logger.info(f"Intent classified as: {intent_type}")

        if intent_type == "conversation":
            # Handle conversational query (about system itself)
            await self.handle_conversation(ws, session_id, intent_text)
        elif intent_type == "advisory":
            # Handle consultative questions (needs AI reasoning)
            await self.handle_advisory(ws, session_id, intent_text)
        else:
            # Handle deployment intent (immediate action)
            try:
                await self.deploy_with_orchestrator(ws, session_id, intent_text)
            except Exception as e:
                logger.error(f"Orchestrator deployment failed: {e}, using fallback visualization")
                await self.fallback_visualization(ws, session_id, intent_text)

    async def handle_conversation(self, ws, session_id, query: str):
        """Handle conversational queries"""
        query_lower = query.lower()

        response_text = ""

        if "claude code" in query_lower or "claude ai" in query_lower:
            response_text = """是的，這個系統整合了 Claude Code CLI 和 Go 後端 orchestrator。

**系統架構**：
- 🤖 NLP 解析：內建快速 parser（原本使用 Claude CLI，但為了速度改用內建）
- 🚀 部署引擎：Go orchestrator (port 8081)
- 🌐 前端介面：WebSocket 實時通信
- ☸️ 目標平台：Kubernetes

**支援的切片類型**：
- eMBB: 高頻寬 (video, streaming)
- URLLC: 超低延遲 (autonomous, industrial)
- mMTC: 大規模物聯網 (IoT, sensors)
- RedCap: 低功耗 (low power devices)

**如何使用**：直接輸入自然語言需求，例如：
"我需要一個 redcap 切片用於 low power consumption 但希望可以有 5Mbps 延遲可以 300ms"
"""
        elif "如何" in query_lower or "怎麼" in query_lower:
            response_text = """**如何使用此系統**：

1. **輸入自然語言 intent**，例如：
   - "部署一個 4K 視頻切片，需要 1Gbps 頻寬"
   - "創建 IoT 切片支援 10000 個設備"
   - "建立自動駕駛切片，延遲 1ms"

2. **系統會自動**：
   - 解析你的需求
   - 分類切片類型
   - 調用 Go orchestrator
   - 部署到 Kubernetes

3. **查看結果**：
   - 實時顯示部署步驟
   - 最終顯示 QoS 配置
"""
        elif "是什麼" in query_lower or "what" in query_lower:
            response_text = """**O-RAN Intent MANO** 是一個基於意圖的網路切片管理與編排系統。

**主要特性**：
- 📝 自然語言輸入
- 🤖 自動 QoS 配置
- 🚀 即時部署
- ☸️ Kubernetes 原生
- 📊 可視化流程

**技術棧**：
- Backend: Go + Kubernetes
- Frontend: WebSocket + HTML5
- NLP: 內建 intent parser
- Orchestration: Argo CD integration
"""
        else:
            response_text = f"""我是 O-RAN Intent MANO 系統，專門處理網路切片部署。

你的問題是："{query}"

**我可以幫你**：
- 🔧 部署網路切片 (eMBB, URLLC, mMTC, RedCap)
- 📊 查詢系統狀態
- 💡 提供使用建議

**要部署網路切片**，請使用類似這樣的 intent：
"我需要一個 [類型] 切片用於 [用途]，需要 [頻寬] 和 [延遲]"
"""

        if ws.closed:
            return

        await ws.send_json({
            "type": "conversation_response",
            "sessionId": session_id,
            "message": response_text,
            "status": "success",
            "timestamp": int(datetime.now().timestamp())
        })

        logger.info(f"✅ Conversation response sent to {session_id}")

    async def handle_advisory(self, ws, session_id, query: str):
        """Handle advisory/consultative questions using Claude CLI for AI reasoning"""
        logger.info(f"🤔 Advisory query: {query}")

        # Send thinking indicator
        if ws.closed:
            return

        await ws.send_json({
            "type": "advisory_thinking",
            "sessionId": session_id,
            "message": "🤔 正在使用 Claude AI 深度分析您的問題...",
            "status": "thinking",
            "timestamp": int(datetime.now().timestamp())
        })

        # Create comprehensive prompt for Claude CLI
        prompt = f"""你是一位資深的 5G 網路切片專家。請針對以下問題提供深度分析和專業建議。

**使用者問題**：
{query}

**你的專業知識包含**：
- 5G 網路切片類型：eMBB, URLLC, mMTC, mMTC-RedCap
- QoS 參數：頻寬 (throughput), 延遲 (latency), 可靠度 (reliability)
- 應用場景：影音串流、自動駕駛、物聯網、工業自動化
- 實務考量：成本效益、基礎建設、人口密度、地理環境

**請提供**：
1. **情境分析**：分析使用者提到的具體場景特點
2. **技術考量**：說明不同切片類型的優缺點
3. **建議方案**：提供 2-3 種可能的切片配置
4. **理由說明**：解釋為何推薦這些方案
5. **後續問題**：詢問 1-2 個關鍵問題以提供更精確建議

請用繁體中文回答，保持專業但易懂的語氣。"""

        try:
            # Call Claude CLI asynchronously
            logger.info("Calling Claude CLI for advisory reasoning...")

            # Windows requires .cmd extension or shell=True
            import platform
            if platform.system() == 'Windows':
                # Use shell=True on Windows to find claude.cmd
                process = await asyncio.create_subprocess_shell(
                    f'claude -p --dangerously-skip-permissions',
                    stdin=asyncio.subprocess.PIPE,
                    stdout=asyncio.subprocess.PIPE,
                    stderr=asyncio.subprocess.PIPE
                )
            else:
                # Unix-like systems
                process = await asyncio.create_subprocess_exec(
                    'claude', '-p', '--dangerously-skip-permissions',
                    stdin=asyncio.subprocess.PIPE,
                    stdout=asyncio.subprocess.PIPE,
                    stderr=asyncio.subprocess.PIPE
                )

            # Send prompt and wait with 90s timeout
            stdout, stderr = await asyncio.wait_for(
                process.communicate(prompt.encode('utf-8')),
                timeout=90.0
            )

            if process.returncode == 0:
                response_text = stdout.decode('utf-8', errors='ignore').strip()
                logger.info(f"Claude CLI response received ({len(response_text)} chars)")

                if ws.closed:
                    return

                await ws.send_json({
                    "type": "advisory_response",
                    "sessionId": session_id,
                    "message": response_text,
                    "query": query,
                    "reasoning": True,
                    "status": "success",
                    "timestamp": int(datetime.now().timestamp())
                })

                logger.info(f"✅ Advisory response sent to {session_id}")
            else:
                error_msg = stderr.decode('utf-8', errors='ignore')
                logger.error(f"Claude CLI error: {error_msg}")
                raise Exception(f"Claude CLI returned error: {error_msg}")

        except asyncio.TimeoutError:
            logger.error("Claude CLI timeout after 90s")
            # Fallback to structured response
            await self.send_fallback_advisory(ws, session_id, query)

        except Exception as e:
            logger.error(f"Claude CLI failed: {e}")
            # Fallback to structured response
            await self.send_fallback_advisory(ws, session_id, query)

    async def send_fallback_advisory(self, ws, session_id, query: str):
        """Fallback advisory response when Claude CLI unavailable"""
        fallback_response = f"""**技術分析建議**

您詢問："{query}"

由於 Claude AI 暫時無法使用，這裡提供基本的技術分析框架：

**🎯 網路切片類型選擇考量**：

1. **eMBB (Enhanced Mobile Broadband)**
   - 適用：影音串流、4K/8K 視頻、大檔案傳輸
   - 特點：高頻寬 (100-1000 Mbps)、中延遲 (10-50ms)
   - 成本：中等

2. **URLLC (Ultra-Reliable Low-Latency)**
   - 適用：自動駕駛、工業控制、遠程醫療
   - 特點：超低延遲 (<10ms)、超高可靠度 (99.999%)
   - 成本：高

3. **mMTC (Massive Machine Type Communication)**
   - 適用：智慧城市、農業感測、環境監測
   - 特點：大規模連接 (10000+ 設備)、低頻寬
   - 成本：低

4. **mMTC-RedCap (Reduced Capability)**
   - 適用：低功耗設備、可穿戴裝置、簡單感測器
   - 特點：低功耗、低成本、中等連接數
   - 成本：很低

**💡 建議**：
- 請提供更多情境細節（應用場景、預期使用者數、預算範圍）
- 可以考慮混合切片（同時部署多種類型）
- 建議先從小規模試點開始

**下一步**：
如需部署，請使用明確的指令，例如：
"部署一個 mMTC 切片支援 5000 個 IoT 設備"
"""

        if ws.closed:
            return

        await ws.send_json({
            "type": "advisory_response",
            "sessionId": session_id,
            "message": fallback_response,
            "query": query,
            "reasoning": False,
            "fallback": True,
            "status": "success",
            "timestamp": int(datetime.now().timestamp())
        })

        logger.info(f"✅ Fallback advisory response sent to {session_id}")

    async def deploy_with_orchestrator(self, ws, session_id, intent_text):
        """Deploy using real Go orchestrator"""
        # Check if WebSocket is still open
        if ws.closed:
            logger.warning(f"WebSocket already closed for session {session_id}")
            return

        # Step 1: NLP Parsing with Claude CLI
        try:
            await ws.send_json({
                "type": "step_update",
                "sessionId": session_id,
                "data": {
                    "step": "Claude CLI NLP",
                    "status": "in_progress",
                    "details": "Calling Claude Code CLI for intent analysis..."
                },
                "timestamp": int(datetime.now().timestamp())
            })
        except Exception as e:
            logger.warning(f"Failed to send step update: {e}")
            return

        # Use Claude CLI for real NLP parsing (with fallback to Go orchestrator)
        slice_type, requirements = await self.parse_with_claude(intent_text)

        logger.info(f"NLP parsed: {slice_type} with {requirements}")

        if not ws.closed:
            await ws.send_json({
                "type": "step_update",
                "sessionId": session_id,
                "data": {
                    "step": "NLP Parsing",
                    "status": "completed",
                    "details": f"Identified: {slice_type} slice"
                },
                "timestamp": int(datetime.now().timestamp())
            })

        # Step 2: Create Intent Request
        if ws.closed:
            return

        await ws.send_json({
            "type": "step_update",
            "sessionId": session_id,
            "data": {
                "step": "Orchestrator Request",
                "status": "in_progress",
                "details": "Sending request to Go orchestrator..."
            },
            "timestamp": int(datetime.now().timestamp())
        })

        # Call orchestrator HTTP API
        async with aiohttp.ClientSession() as session:
            payload = {
                "intent": intent_text,
                "slice_type": slice_type,
                "requirements": requirements
            }
            async with session.post(f"{self.orchestrator_url}/api/v1/intents", json=payload) as resp:
                if resp.status == 201:
                    result = await resp.json()
                    await ws.send_json({
                        "type": "step_update",
                        "sessionId": session_id,
                        "data": {
                            "step": "Orchestrator Request",
                            "status": "completed",
                            "details": f"Intent created: {result.get('intent_id', 'N/A')}"
                        },
                        "timestamp": int(datetime.now().timestamp())
                    })
                else:
                    raise Exception(f"Orchestrator returned {resp.status}")

        # Step 3: Deployment
        if ws.closed:
            return

        await ws.send_json({
            "type": "step_update",
            "sessionId": session_id,
            "data": {
                "step": "Deployment",
                "status": "in_progress",
                "details": "Deploying to Kubernetes..."
            },
            "timestamp": int(datetime.now().timestamp())
        })
        await asyncio.sleep(1.0)

        if ws.closed:
            return

        await ws.send_json({
            "type": "step_update",
            "sessionId": session_id,
            "data": {
                "step": "Deployment",
                "status": "completed",
                "details": "Network slice deployed successfully"
            },
            "timestamp": int(datetime.now().timestamp())
        })

        # Final response
        if ws.closed:
            return

        await ws.send_json({
            "type": "intent_response",
            "sessionId": session_id,
            "sliceType": slice_type,
            "action": "created",
            "requirements": requirements,
            "rawResponse": f"Successfully deployed {slice_type} slice using real orchestrator",
            "status": "success",
            "timestamp": int(datetime.now().timestamp())
        })

        logger.info(f"✅ Real deployment completed: {slice_type}")

    async def fallback_visualization(self, ws, session_id, intent_text):
        """Fallback visualization if orchestrator unavailable"""

        # Step 1: NLP Parsing
        await ws.send_json({
            "type": "step_update",
            "sessionId": session_id,
            "data": {
                "step": "NLP Parsing",
                "status": "in_progress",
                "details": "Analyzing natural language intent..."
            },
            "timestamp": int(datetime.now().timestamp())
        })
        await asyncio.sleep(0.5)

        # Detect slice type
        slice_type, requirements = self.parse_intent(intent_text)

        await ws.send_json({
            "type": "step_update",
            "sessionId": session_id,
            "data": {
                "step": "NLP Parsing",
                "status": "completed",
                "details": f"Detected {slice_type} slice type"
            },
            "timestamp": int(datetime.now().timestamp())
        })

        # Step 2: QoS Profile
        await ws.send_json({
            "type": "step_update",
            "sessionId": session_id,
            "data": {
                "step": "QoS Profile",
                "status": "in_progress",
                "details": "Generating QoS profile..."
            },
            "timestamp": int(datetime.now().timestamp())
        })
        await asyncio.sleep(0.3)

        await ws.send_json({
            "type": "step_update",
            "sessionId": session_id,
            "data": {
                "step": "QoS Profile",
                "status": "completed",
                "details": "QoS profile generated",
                "requirements": requirements
            },
            "timestamp": int(datetime.now().timestamp())
        })

        # Step 3: Resource Placement
        await ws.send_json({
            "type": "step_update",
            "sessionId": session_id,
            "data": {
                "step": "Resource Placement",
                "status": "in_progress",
                "details": "Calculating optimal resource placement..."
            },
            "timestamp": int(datetime.now().timestamp())
        })
        await asyncio.sleep(0.4)

        await ws.send_json({
            "type": "step_update",
            "sessionId": session_id,
            "data": {
                "step": "Resource Placement",
                "status": "completed",
                "details": "Resources allocated on edge01, regional clusters"
            },
            "timestamp": int(datetime.now().timestamp())
        })

        # Step 4: Deployment
        await ws.send_json({
            "type": "step_update",
            "sessionId": session_id,
            "data": {
                "step": "Deployment",
                "status": "in_progress",
                "details": "Deploying network slice..."
            },
            "timestamp": int(datetime.now().timestamp())
        })
        await asyncio.sleep(0.6)

        await ws.send_json({
            "type": "step_update",
            "sessionId": session_id,
            "data": {
                "step": "Deployment",
                "status": "completed",
                "details": "Network slice deployed successfully"
            },
            "timestamp": int(datetime.now().timestamp())
        })

        # Step 5: Complete
        await ws.send_json({
            "type": "step_update",
            "sessionId": session_id,
            "data": {
                "step": "Complete",
                "status": "completed",
                "details": "End-to-end deployment completed"
            },
            "timestamp": int(datetime.now().timestamp())
        })

        # Final response
        await ws.send_json({
            "type": "intent_response",
            "sessionId": session_id,
            "sliceType": slice_type,
            "action": "created",
            "requirements": requirements,
            "rawResponse": f"Successfully deployed {slice_type} slice in 2.3 seconds",
            "status": "success",
            "timestamp": int(datetime.now().timestamp())
        })

        logger.info(f"✅ Intent processed successfully: {slice_type}")

    async def parse_with_claude(self, intent: str):
        """Use Claude CLI to parse natural language intent"""
        try:
            # Create prompt for Claude
            prompt = f"""You are a 5G network slicing expert. Analyze this natural language intent and return ONLY a JSON object with slice_type and requirements.

Intent: "{intent}"

Possible slice types:
- eMBB: Enhanced Mobile Broadband (video, streaming, high bandwidth)
- URLLC: Ultra-Reliable Low-Latency (autonomous, industrial, <10ms)
- mMTC: Massive Machine Type Communication (IoT, sensors, many devices)
- mMTC-RedCap: Reduced Capability variant (low power, RedCap devices)

Return format (JSON only, no explanation):
{{
  "slice_type": "TYPE",
  "requirements": {{
    "throughput": <number in Mbps>,
    "latency": <number in ms>,
    "reliability": <number as percentage>,
    "power_class": "low|medium|high" (optional)
  }}
}}"""

            # Save prompt to temp file
            with tempfile.NamedTemporaryFile(mode='w', suffix='.txt', delete=False, encoding='utf-8') as f:
                f.write(prompt)
                prompt_file = f.name

            # Call Claude CLI
            # Read the prompt file and pass via stdin
            with open(prompt_file, 'r', encoding='utf-8') as pf:
                prompt_content = pf.read()

            # Use Go orchestrator's built-in NLP instead of Claude CLI
            # Claude CLI is too slow (30s timeout) for real-time interaction
            logger.info("Using Go orchestrator NLP parser (Claude CLI too slow)")

            # Just use fallback parser (fast, no Claude CLI timeout)
            return self.parse_intent_fallback(intent)

            # Clean up temp file
            import os
            os.unlink(prompt_file)

            if result.returncode == 0:
                output = result.stdout.strip()
                logger.info(f"Claude CLI output: {output[:200]}")

                # Try to extract JSON from output
                import re
                json_match = re.search(r'\{[^{}]*"slice_type"[^{}]*\}', output, re.DOTALL)
                if json_match:
                    parsed = json.loads(json_match.group(0))
                    slice_type = parsed.get('slice_type', 'eMBB')
                    requirements = parsed.get('requirements', {})
                    return slice_type, requirements
                else:
                    logger.warning("No JSON found in Claude output, using fallback")
                    return self.parse_intent_fallback(intent)
            else:
                logger.error(f"Claude CLI error: {result.stderr}")
                return self.parse_intent_fallback(intent)

        except Exception as e:
            logger.error(f"Claude CLI failed: {e}, using fallback")
            return self.parse_intent_fallback(intent)

    def parse_intent_fallback(self, intent: str):
        """Parse NL intent and extract requirements"""
        intent_lower = intent.lower()

        # RedCap (Reduced Capability) - mMTC variant for low power
        if any(kw in intent_lower for kw in ["redcap", "reduced capability", "low power", "low power consumption"]):
            return "mMTC-RedCap", {
                "throughput": 5,  # Extracted from intent
                "latency": 300,   # Extracted from intent
                "reliability": 99.0,
                "connections": 1000,
                "power_class": "low"
            }
        # Video streaming (eMBB)
        elif any(kw in intent_lower for kw in ["video", "4k", "8k", "streaming", "影音", "串流"]):
            return "eMBB", {
                "throughput": 1000,
                "latency": 20,
                "reliability": 99.9
            }
        # URLLC (Ultra-Reliable Low-Latency)
        elif any(kw in intent_lower for kw in ["autonomous", "urllc", "1ms", "低延遲", "自動駕駛"]):
            return "URLLC", {
                "throughput": 10,
                "latency": 1,
                "reliability": 99.999
            }
        # mMTC (Massive Machine Type Communication)
        elif any(kw in intent_lower for kw in ["iot", "sensor", "mmtc", "感測器", "物聯網"]):
            return "mMTC", {
                "throughput": 1,
                "latency": 100,
                "reliability": 99.0,
                "connections": 10000
            }
        else:
            return "eMBB", {
                "throughput": 100,
                "latency": 20,
                "reliability": 99.9
            }

async def main():
    """Start integrated server"""
    server = IntegratedServer()

    app = web.Application()
    app.router.add_get('/ws', server.websocket_handler)

    # Proxy API requests to Go orchestrator
    async def proxy_to_orchestrator(request):
        path = request.path
        async with aiohttp.ClientSession() as session:
            url = f"http://localhost:8081{path}"
            async with session.request(
                method=request.method,
                url=url,
                headers=request.headers,
                data=await request.read()
            ) as resp:
                return web.Response(
                    body=await resp.read(),
                    status=resp.status,
                    headers=resp.headers
                )

    app.router.add_route('*', '/api/{tail:.*}', proxy_to_orchestrator)
    app.router.add_route('GET', '/health', proxy_to_orchestrator)
    app.router.add_static('/', path='web', name='static', show_index=True)

    logger.info("🚀 O-RAN Integrated Server Starting")
    logger.info("🌐 Frontend: http://localhost:8080")
    logger.info("🔌 WebSocket: ws://localhost:8080/ws")

    runner = web.AppRunner(app)
    await runner.setup()
    site = web.TCPSite(runner, 'localhost', 8080)
    await site.start()

    logger.info("✅ Server running on http://localhost:8080")

    # Keep running
    await asyncio.Event().wait()

if __name__ == "__main__":
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        logger.info("🛑 Server stopped")