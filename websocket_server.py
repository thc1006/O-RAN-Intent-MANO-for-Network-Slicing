#!/usr/bin/env python3
"""
O-RAN Intent MANO WebSocket Server
Quick implementation for E2E testing
"""

import asyncio
import json
import logging
from datetime import datetime
from typing import Dict, Any
import websockets
from http.server import HTTPServer, SimpleHTTPRequestHandler
import threading
import os

logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

class WebSocketServer:
    def __init__(self):
        self.clients = {}

    async def handle_client(self, websocket):
        """Handle WebSocket client connection"""
        session_id = f"session-{datetime.now().strftime('%Y%m%d%H%M%S')}"
        self.clients[session_id] = websocket
        logger.info(f"✅ New client connected: {session_id}")

        try:
            # Send welcome message
            await websocket.send(json.dumps({
                "type": "connected",
                "sessionId": session_id,
                "message": "Connected to O-RAN Network Slicing Service",
                "status": "success",
                "timestamp": int(datetime.now().timestamp())
            }))

            # Handle messages
            async for message in websocket:
                await self.process_message(websocket, session_id, message)

        except websockets.exceptions.ConnectionClosed:
            logger.info(f"❌ Client disconnected: {session_id}")
        finally:
            del self.clients[session_id]

    async def process_message(self, websocket, session_id, message):
        """Process incoming message"""
        try:
            data = json.loads(message)
            intent_type = data.get("type", "")

            logger.info(f"📥 Received {intent_type} from {session_id}")

            if intent_type == "intent":
                await self.process_intent(websocket, session_id, data)
            elif intent_type == "ping":
                await websocket.send(json.dumps({"type": "pong"}))

        except json.JSONDecodeError as e:
            logger.error(f"JSON decode error: {e}")

    async def process_intent(self, websocket, session_id, data):
        """Process NL intent with visualization"""
        intent_text = data.get("intent", "")
        logger.info(f"🎯 Processing intent: {intent_text}")

        # Step 1: NLP Parsing
        await websocket.send(json.dumps({
            "type": "step_update",
            "sessionId": session_id,
            "data": {
                "step": "NLP Parsing",
                "status": "in_progress",
                "details": "Analyzing natural language intent..."
            },
            "timestamp": int(datetime.now().timestamp())
        }))
        await asyncio.sleep(0.5)

        # Detect slice type
        slice_type, requirements = self.parse_intent(intent_text)

        await websocket.send(json.dumps({
            "type": "step_update",
            "sessionId": session_id,
            "data": {
                "step": "NLP Parsing",
                "status": "completed",
                "details": f"Detected {slice_type} slice type"
            },
            "timestamp": int(datetime.now().timestamp())
        }))

        # Step 2: QoS Profile Generation
        await websocket.send(json.dumps({
            "type": "step_update",
            "sessionId": session_id,
            "data": {
                "step": "QoS Profile",
                "status": "in_progress",
                "details": "Generating QoS profile..."
            },
            "timestamp": int(datetime.now().timestamp())
        }))
        await asyncio.sleep(0.3)

        await websocket.send(json.dumps({
            "type": "step_update",
            "sessionId": session_id,
            "data": {
                "step": "QoS Profile",
                "status": "completed",
                "details": "QoS profile generated",
                "requirements": requirements
            },
            "timestamp": int(datetime.now().timestamp())
        }))

        # Step 3: Resource Placement
        await websocket.send(json.dumps({
            "type": "step_update",
            "sessionId": session_id,
            "data": {
                "step": "Resource Placement",
                "status": "in_progress",
                "details": "Calculating optimal resource placement..."
            },
            "timestamp": int(datetime.now().timestamp())
        }))
        await asyncio.sleep(0.4)

        await websocket.send(json.dumps({
            "type": "step_update",
            "sessionId": session_id,
            "data": {
                "step": "Resource Placement",
                "status": "completed",
                "details": "Resources allocated on edge01, regional clusters"
            },
            "timestamp": int(datetime.now().timestamp())
        }))

        # Step 4: Deployment
        await websocket.send(json.dumps({
            "type": "step_update",
            "sessionId": session_id,
            "data": {
                "step": "Deployment",
                "status": "in_progress",
                "details": "Deploying network slice..."
            },
            "timestamp": int(datetime.now().timestamp())
        }))
        await asyncio.sleep(0.6)

        await websocket.send(json.dumps({
            "type": "step_update",
            "sessionId": session_id,
            "data": {
                "step": "Deployment",
                "status": "completed",
                "details": "Network slice deployed successfully"
            },
            "timestamp": int(datetime.now().timestamp())
        }))

        # Step 5: Complete
        await websocket.send(json.dumps({
            "type": "step_update",
            "sessionId": session_id,
            "data": {
                "step": "Complete",
                "status": "completed",
                "details": "End-to-end deployment completed"
            },
            "timestamp": int(datetime.now().timestamp())
        }))

        # Final response
        await websocket.send(json.dumps({
            "type": "intent_response",
            "sessionId": session_id,
            "sliceType": slice_type,
            "action": "created",
            "requirements": requirements,
            "rawResponse": f"Successfully deployed {slice_type} slice in 2.3 seconds",
            "status": "success",
            "timestamp": int(datetime.now().timestamp())
        }))

        logger.info(f"✅ Intent processed: {slice_type}")

    def parse_intent(self, intent: str) -> tuple:
        """Parse NL intent and extract requirements"""
        intent_lower = intent.lower()

        # Default to eMBB
        slice_type = "eMBB"
        requirements = {
            "throughput": 100,
            "latency": 20,
            "reliability": 99.9
        }

        # Detect slice type
        if any(keyword in intent_lower for keyword in ["video", "4k", "8k", "streaming", "影音", "串流"]):
            slice_type = "eMBB"
            requirements = {
                "throughput": 1000,
                "latency": 20,
                "reliability": 99.9
            }
        elif any(keyword in intent_lower for keyword in ["autonomous", "urllc", "1ms", "低延遲", "自動駕駛"]):
            slice_type = "URLLC"
            requirements = {
                "throughput": 10,
                "latency": 1,
                "reliability": 99.999
            }
        elif any(keyword in intent_lower for keyword in ["iot", "sensor", "mmtc", "感測器", "物聯網"]):
            slice_type = "mMTC"
            requirements = {
                "throughput": 1,
                "latency": 100,
                "reliability": 99.0,
                "connections": 10000
            }

        return slice_type, requirements

async def main():
    """Start WebSocket server"""
    server = WebSocketServer()

    logger.info("🚀 Starting O-RAN WebSocket Server")
    logger.info("📡 WebSocket endpoint: ws://localhost:8081/ws")
    logger.info("🌐 Frontend: http://localhost:8080")

    async with websockets.serve(
        lambda ws: server.handle_client(ws),
        "localhost",
        8081
    ):
        logger.info("✅ WebSocket server running on port 8081")
        await asyncio.Future()  # run forever

if __name__ == "__main__":
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        logger.info("🛑 Server stopped")