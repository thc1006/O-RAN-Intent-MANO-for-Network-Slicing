#!/usr/bin/env python3
"""
NLP Service for O-RAN Intent-Based MANO
FastAPI HTTP service wrapping intent_parser.py
Provides REST API for natural language intent processing
"""

from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel, Field
from typing import Dict, Any, Optional
import logging
import uvicorn
from datetime import datetime

# Import the intent parser
from intent_parser import IntentParser, IntentValidationError, QoSMapping

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# Create FastAPI app
app = FastAPI(
    title="O-RAN NLP Service",
    description="Natural Language Processing service for O-RAN Intent-Based MANO",
    version="1.0.0",
    docs_url="/api/docs",
    redoc_url="/api/redoc"
)

# Add CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],  # In production, specify exact origins
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Global intent parser instance
parser = IntentParser()

# Pydantic models for API
class IntentRequest(BaseModel):
    """Request model for intent parsing"""
    intent: str = Field(..., min_length=1, max_length=1000, description="Natural language intent")
    session_id: Optional[str] = Field(None, description="Optional session ID for tracking")

    class Config:
        schema_extra = {
            "example": {
                "intent": "Deploy high-bandwidth video streaming slice for 100 concurrent users",
                "session_id": "session-20250101-001"
            }
        }

class QoSResponse(BaseModel):
    """Response model for parsed QoS parameters"""
    slice_type: str
    throughput_mbps: float
    latency_ms: float
    packet_loss_rate: float
    priority: int
    jitter_ms: Optional[float] = None
    bandwidth_guarantee: Optional[float] = None
    reliability: Optional[float] = None

class IntentResponse(BaseModel):
    """Response model for intent parsing"""
    success: bool
    slice_type: str
    qos_profile: QoSResponse
    raw_intent: str
    session_id: Optional[str] = None
    timestamp: str
    processing_time_ms: float

    class Config:
        schema_extra = {
            "example": {
                "success": True,
                "slice_type": "eMBB",
                "qos_profile": {
                    "slice_type": "eMBB",
                    "throughput_mbps": 50.0,
                    "latency_ms": 10.0,
                    "packet_loss_rate": 0.001,
                    "priority": 5,
                    "reliability": 0.999
                },
                "raw_intent": "Deploy high-bandwidth video streaming slice",
                "session_id": "session-20250101-001",
                "timestamp": "2025-10-01T00:45:00Z",
                "processing_time_ms": 12.5
            }
        }

class ErrorResponse(BaseModel):
    """Error response model"""
    success: bool = False
    error: str
    detail: Optional[str] = None
    timestamp: str

class HealthResponse(BaseModel):
    """Health check response"""
    status: str
    version: str
    uptime_seconds: float
    total_intents_processed: int

# Metrics
start_time = datetime.now()
total_intents = 0

@app.get("/health", response_model=HealthResponse)
async def health_check():
    """Health check endpoint"""
    uptime = (datetime.now() - start_time).total_seconds()
    return HealthResponse(
        status="healthy",
        version="1.0.0",
        uptime_seconds=uptime,
        total_intents_processed=total_intents
    )

@app.post("/api/v1/parse", response_model=IntentResponse)
async def parse_intent(request: IntentRequest):
    """
    Parse natural language intent into QoS parameters

    Args:
        request: IntentRequest with natural language text

    Returns:
        IntentResponse with parsed QoS parameters

    Raises:
        HTTPException: If parsing fails
    """
    global total_intents

    start = datetime.now()
    logger.info(f"Parsing intent: {request.intent[:50]}...")

    try:
        # Parse the intent
        mapping: QoSMapping = parser.parse(request.intent)

        # Create QoS response
        qos_response = QoSResponse(
            slice_type=mapping.slice_type.value,
            throughput_mbps=mapping.throughput_mbps,
            latency_ms=mapping.latency_ms,
            packet_loss_rate=mapping.packet_loss_rate,
            priority=mapping.priority,
            jitter_ms=mapping.jitter_ms,
            bandwidth_guarantee=mapping.bandwidth_guarantee,
            reliability=mapping.reliability
        )

        # Calculate processing time
        processing_time = (datetime.now() - start).total_seconds() * 1000

        # Update metrics
        total_intents += 1

        logger.info(f"✓ Parsed as {mapping.slice_type.value} in {processing_time:.2f}ms")

        return IntentResponse(
            success=True,
            slice_type=mapping.slice_type.value,
            qos_profile=qos_response,
            raw_intent=request.intent,
            session_id=request.session_id,
            timestamp=datetime.now().isoformat(),
            processing_time_ms=processing_time
        )

    except IntentValidationError as e:
        logger.error(f"Validation error: {e}")
        raise HTTPException(
            status_code=400,
            detail={
                "success": False,
                "error": "Intent validation failed",
                "detail": str(e),
                "timestamp": datetime.now().isoformat()
            }
        )
    except ValueError as e:
        logger.error(f"Value error: {e}")
        raise HTTPException(
            status_code=400,
            detail={
                "success": False,
                "error": "Invalid intent format",
                "detail": str(e),
                "timestamp": datetime.now().isoformat()
            }
        )
    except Exception as e:
        logger.error(f"Unexpected error: {e}", exc_info=True)
        raise HTTPException(
            status_code=500,
            detail={
                "success": False,
                "error": "Internal server error",
                "detail": str(e),
                "timestamp": datetime.now().isoformat()
            }
        )

@app.get("/api/v1/history")
async def get_history():
    """Get history of parsed intents"""
    history = parser.get_history()
    return {
        "success": True,
        "count": len(history),
        "intents": [
            {
                "intent": intent,
                "slice_type": mapping.slice_type.value,
                "throughput_mbps": mapping.throughput_mbps,
                "latency_ms": mapping.latency_ms
            }
            for intent, mapping in history
        ]
    }

@app.get("/")
async def root():
    """Root endpoint"""
    return {
        "service": "O-RAN NLP Service",
        "version": "1.0.0",
        "status": "running",
        "docs": "/api/docs"
    }

if __name__ == "__main__":
    logger.info("🚀 Starting O-RAN NLP Service")
    logger.info("📡 API Documentation: http://localhost:8082/api/docs")
    logger.info("🔍 Health Check: http://localhost:8082/health")

    uvicorn.run(
        app,
        host="0.0.0.0",
        port=8082,
        log_level="info"
    )
