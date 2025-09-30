#!/usr/bin/env python3
"""
Tests for NLP Service API
"""

import pytest
from fastapi.testclient import TestClient
from nlp_service import app

client = TestClient(app)

def test_health_check():
    """Test health check endpoint"""
    response = client.get("/health")
    assert response.status_code == 200
    data = response.json()
    assert data["status"] == "healthy"
    assert "version" in data
    assert "uptime_seconds" in data

def test_root_endpoint():
    """Test root endpoint"""
    response = client.get("/")
    assert response.status_code == 200
    data = response.json()
    assert data["service"] == "O-RAN NLP Service"
    assert data["status"] == "running"

def test_parse_embb_intent():
    """Test parsing eMBB intent"""
    response = client.post(
        "/api/v1/parse",
        json={
            "intent": "Deploy high-bandwidth video streaming slice for 100 users",
            "session_id": "test-session-001"
        }
    )
    assert response.status_code == 200
    data = response.json()
    assert data["success"] is True
    assert data["slice_type"] == "eMBB"
    assert data["qos_profile"]["throughput_mbps"] == 50.0
    assert data["qos_profile"]["latency_ms"] == 10.0
    assert data["session_id"] == "test-session-001"

def test_parse_urllc_intent():
    """Test parsing URLLC intent"""
    response = client.post(
        "/api/v1/parse",
        json={
            "intent": "Deploy ultra-low latency slice for autonomous vehicles"
        }
    )
    assert response.status_code == 200
    data = response.json()
    assert data["success"] is True
    assert data["slice_type"] == "URLLC"
    assert data["qos_profile"]["latency_ms"] == 1.0
    assert data["qos_profile"]["reliability"] == 0.99999

def test_parse_mmtc_intent():
    """Test parsing mMTC intent"""
    response = client.post(
        "/api/v1/parse",
        json={
            "intent": "Deploy IoT sensor network for smart city monitoring"
        }
    )
    assert response.status_code == 200
    data = response.json()
    assert data["success"] is True
    assert data["slice_type"] == "mMTC"

def test_empty_intent():
    """Test empty intent validation"""
    response = client.post(
        "/api/v1/parse",
        json={"intent": ""}
    )
    # FastAPI/Pydantic returns 422 for validation errors
    assert response.status_code == 422

def test_invalid_intent():
    """Test invalid intent (only numbers)"""
    response = client.post(
        "/api/v1/parse",
        json={"intent": "12345 !@#$%"}
    )
    assert response.status_code == 400

def test_get_history():
    """Test history endpoint"""
    # First parse an intent
    client.post(
        "/api/v1/parse",
        json={"intent": "Deploy video streaming slice"}
    )

    # Get history
    response = client.get("/api/v1/history")
    assert response.status_code == 200
    data = response.json()
    assert data["success"] is True
    assert "intents" in data
    assert len(data["intents"]) > 0

def test_processing_time():
    """Test that processing time is tracked"""
    response = client.post(
        "/api/v1/parse",
        json={"intent": "Deploy high-speed data transfer slice"}
    )
    assert response.status_code == 200
    data = response.json()
    assert "processing_time_ms" in data
    assert data["processing_time_ms"] > 0

if __name__ == "__main__":
    pytest.main([__file__, "-v"])
