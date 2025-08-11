from fastapi import FastAPI, Request, HTTPException
from pydantic import BaseModel, Field
from typing import Any, Dict, List, Optional, Union
import uvicorn
import torch
import torch.nn as nn
import numpy as np
import logging
import os
import json
from transformers import AutoTokenizer, AutoModel
from pathlib import Path
import asyncio

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

app = FastAPI(
    title="OpenShift AI Inference Server",
    description="ML Inference API for OpenShift AI with CI/CD automation",
    version="1.0.0"
)

class InferenceRequest(BaseModel):
    inputs: Union[List[str], List[List[float]], str, List[float]] = Field(..., description="Input data for inference")
    model_name: Optional[str] = Field(default="default", description="Name of the model to use")
    parameters: Optional[Dict[str, Any]] = Field(default_factory=dict, description="Additional parameters")

class InferenceResponse(BaseModel):
    outputs: Any = Field(..., description="Model outputs")
    model_name: str = Field(..., description="Name of the model used")
    processing_time_ms: float = Field(..., description="Processing time in milliseconds")
    metadata: Optional[Dict[str, Any]] = Field(default_factory=dict, description="Additional metadata")

class ModelRegistry:
    def __init__(self):
        self.models = {}
        self.tokenizers = {}
        self._load_default_models()
    
    def _load_default_models(self):
        """Load default models"""
        try:
            # Simple neural network for demonstration
            class SimpleModel(nn.Module):
                def __init__(self, input_size=10, hidden_size=50, output_size=1):
                    super(SimpleModel, self).__init__()
                    self.network = nn.Sequential(
                        nn.Linear(input_size, hidden_size),
                        nn.ReLU(),
                        nn.Linear(hidden_size, hidden_size),
                        nn.ReLU(),
                        nn.Linear(hidden_size, output_size)
                    )
                
                def forward(self, x):
                    return self.network(x)
            
            self.models["default"] = SimpleModel()
            self.models["simple_classifier"] = SimpleModel(input_size=4, output_size=3)
            
            # Try to load a text model (smaller one for demo)
            try:
                tokenizer = AutoTokenizer.from_pretrained("sentence-transformers/all-MiniLM-L6-v2")
                model = AutoModel.from_pretrained("sentence-transformers/all-MiniLM-L6-v2")
                self.models["text_embeddings"] = model
                self.tokenizers["text_embeddings"] = tokenizer
                logger.info("Loaded text embedding model")
            except Exception as e:
                logger.warning(f"Could not load text model: {e}")
                
        except Exception as e:
            logger.error(f"Error loading default models: {e}")
    
    def get_model(self, model_name: str):
        return self.models.get(model_name)
    
    def get_tokenizer(self, model_name: str):
        return self.tokenizers.get(model_name)
    
    def list_models(self):
        return list(self.models.keys())

# Global model registry
model_registry = ModelRegistry()

class InferenceEngine:
    @staticmethod
    async def predict(inputs: Any, model_name: str, parameters: Dict[str, Any] = None) -> Any:
        """Perform inference with the specified model"""
        if parameters is None:
            parameters = {}
            
        model = model_registry.get_model(model_name)
        if model is None:
            raise HTTPException(status_code=404, f"Model '{model_name}' not found")
        
        try:
            if model_name == "text_embeddings":
                return await InferenceEngine._text_embedding_inference(inputs, model, model_name)
            else:
                return await InferenceEngine._numeric_inference(inputs, model)
        except Exception as e:
            logger.error(f"Inference error: {e}")
            raise HTTPException(status_code=500, detail=f"Inference failed: {str(e)}")
    
    @staticmethod
    async def _text_embedding_inference(inputs: Union[str, List[str]], model, model_name: str):
        """Handle text embedding inference"""
        tokenizer = model_registry.get_tokenizer(model_name)
        if tokenizer is None:
            raise ValueError("Tokenizer not found for text model")
        
        if isinstance(inputs, str):
            inputs = [inputs]
        
        # Tokenize and encode
        encoded = tokenizer(inputs, padding=True, truncation=True, return_tensors="pt")
        
        with torch.no_grad():
            outputs = model(**encoded)
            # Use mean pooling for sentence embeddings
            embeddings = outputs.last_hidden_state.mean(dim=1)
        
        return embeddings.numpy().tolist()
    
    @staticmethod
    async def _numeric_inference(inputs: Union[List[float], List[List[float]]], model):
        """Handle numeric inference"""
        # Convert inputs to tensor
        if isinstance(inputs[0], (int, float)):
            # Single sample
            tensor_input = torch.tensor([inputs], dtype=torch.float32)
        else:
            # Batch of samples
            tensor_input = torch.tensor(inputs, dtype=torch.float32)
        
        with torch.no_grad():
            outputs = model(tensor_input)
        
        return outputs.numpy().tolist()

@app.post("/infer", response_model=InferenceResponse)
async def infer(request: InferenceRequest):
    """Main inference endpoint"""
    import time
    start_time = time.time()
    
    try:
        outputs = await InferenceEngine.predict(
            request.inputs, 
            request.model_name, 
            request.parameters
        )
        
        processing_time = (time.time() - start_time) * 1000
        
        return InferenceResponse(
            outputs=outputs,
            model_name=request.model_name,
            processing_time_ms=processing_time,
            metadata={"input_shape": str(np.array(request.inputs).shape) if isinstance(request.inputs, list) else "text"}
        )
    
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Unexpected error during inference: {e}")
        raise HTTPException(status_code=500, detail="Internal server error")

@app.get("/models")
def list_models():
    """List available models"""
    return {
        "models": model_registry.list_models(),
        "count": len(model_registry.list_models())
    }

@app.get("/health")
def health():
    """Health check endpoint"""
    return {
        "status": "healthy",
        "service": "openshift-ai-inference-server",
        "models_loaded": len(model_registry.list_models())
    }

@app.get("/")
def root():
    """Root endpoint with service information"""
    return {
        "service": "OpenShift AI Inference Server",
        "version": "1.0.0",
        "description": "ML Inference API with CI/CD automation capabilities",
        "endpoints": {
            "inference": "/infer",
            "models": "/models", 
            "health": "/health",
            "docs": "/docs"
        }
    }

if __name__ == "__main__":
    uvicorn.run(
        app, 
        host="0.0.0.0", 
        port=int(os.environ.get("PORT", 8080)),
        log_level="info"
    )
