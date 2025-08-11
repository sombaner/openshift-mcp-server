"""
Minimal inference server without heavy ML dependencies
Focused on CI/CD integration and lightweight responses
"""

from fastapi import FastAPI, Request, HTTPException
from pydantic import BaseModel
from typing import Dict, List, Any, Optional
import json
import random
import time

app = FastAPI(
    title="OpenShift AI MCP Server - Minimal Inference",
    description="Lightweight inference server optimized for CI/CD workflows",
    version="1.0.0"
)

# Request/Response models
class InferenceRequest(BaseModel):
    inputs: Any
    model_name: Optional[str] = "lightweight"
    parameters: Optional[Dict[str, Any]] = {}

class InferenceResponse(BaseModel):
    outputs: Any
    model_name: str
    processing_time_ms: float
    metadata: Dict[str, Any]

class ModelInfo(BaseModel):
    name: str
    type: str
    description: str
    size_mb: float
    ready: bool

class HealthResponse(BaseModel):
    status: str
    service: str
    models_loaded: int
    uptime_seconds: float

# Simple in-memory model registry (no actual ML models)
LIGHTWEIGHT_MODELS = {
    "lightweight": ModelInfo(
        name="lightweight",
        type="mock",
        description="Lightweight mock model for CI/CD testing",
        size_mb=0.1,
        ready=True
    ),
    "text-classifier": ModelInfo(
        name="text-classifier",
        type="classification",
        description="Simple text classification mock",
        size_mb=0.2,
        ready=True
    ),
    "sentiment-analyzer": ModelInfo(
        name="sentiment-analyzer", 
        type="sentiment",
        description="Basic sentiment analysis mock",
        size_mb=0.1,
        ready=True
    )
}

# Service start time for uptime calculation
START_TIME = time.time()

@app.get("/")
async def root():
    """Root endpoint with service information"""
    return {
        "service": "OpenShift AI MCP Server - Minimal Inference",
        "version": "1.0.0",
        "description": "Lightweight inference server optimized for CI/CD workflows",
        "endpoints": {
            "inference": "/infer",
            "models": "/models", 
            "health": "/health"
        },
        "features": [
            "Fast startup time",
            "Small memory footprint",
            "CI/CD optimized",
            "Container-friendly"
        ]
    }

@app.post("/infer", response_model=InferenceResponse)
async def inference(request: InferenceRequest):
    """
    Lightweight inference endpoint with mock responses
    Optimized for testing CI/CD pipelines without heavy ML processing
    """
    start_time = time.time()
    
    model_name = request.model_name or "lightweight"
    
    if model_name not in LIGHTWEIGHT_MODELS:
        raise HTTPException(status_code=404, detail=f"Model {model_name} not found")
    
    model = LIGHTWEIGHT_MODELS[model_name]
    if not model.ready:
        raise HTTPException(status_code=503, detail=f"Model {model_name} not ready")
    
    # Generate mock responses based on model type
    if model.type == "classification":
        outputs = {
            "predictions": ["positive", "negative", "neutral"][random.randint(0, 2)],
            "confidence": round(random.uniform(0.7, 0.99), 3),
            "categories": ["positive", "negative", "neutral"]
        }
    elif model.type == "sentiment":
        sentiment_score = random.uniform(-1, 1)
        outputs = {
            "sentiment": "positive" if sentiment_score > 0 else "negative",
            "score": round(sentiment_score, 3),
            "confidence": round(random.uniform(0.8, 0.95), 3)
        }
    else:
        # Default lightweight response
        outputs = {
            "message": "Mock inference completed successfully",
            "input_processed": True,
            "response_id": f"resp_{int(time.time())}"
        }
    
    processing_time = (time.time() - start_time) * 1000
    
    return InferenceResponse(
        outputs=outputs,
        model_name=model_name,
        processing_time_ms=round(processing_time, 2),
        metadata={
            "model_type": model.type,
            "model_size_mb": model.size_mb,
            "timestamp": time.time(),
            "mode": "mock"
        }
    )

@app.get("/models")
async def list_models():
    """List available lightweight models"""
    return {
        "models": list(LIGHTWEIGHT_MODELS.values()),
        "total_models": len(LIGHTWEIGHT_MODELS),
        "total_size_mb": sum(model.size_mb for model in LIGHTWEIGHT_MODELS.values())
    }

@app.get("/models/{model_name}")
async def get_model(model_name: str):
    """Get specific model information"""
    if model_name not in LIGHTWEIGHT_MODELS:
        raise HTTPException(status_code=404, detail=f"Model {model_name} not found")
    
    return LIGHTWEIGHT_MODELS[model_name]

@app.get("/health", response_model=HealthResponse)
async def health_check():
    """Health check endpoint"""
    uptime = time.time() - START_TIME
    
    return HealthResponse(
        status="healthy",
        service="minimal-inference-server",
        models_loaded=len(LIGHTWEIGHT_MODELS),
        uptime_seconds=round(uptime, 2)
    )

@app.get("/metrics")
async def metrics():
    """Basic metrics for monitoring"""
    uptime = time.time() - START_TIME
    
    return {
        "uptime_seconds": round(uptime, 2),
        "models_available": len(LIGHTWEIGHT_MODELS),
        "memory_usage": "minimal",
        "status": "healthy",
        "build_info": {
            "optimized": True,
            "ml_libraries": "none",
            "size_optimized": True
        }
    }

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8080)
