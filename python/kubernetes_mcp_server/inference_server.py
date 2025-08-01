from fastapi import FastAPI, Request
from pydantic import BaseModel
from typing import Any
import uvicorn

app = FastAPI()

class InferenceRequest(BaseModel):
    input: Any

class InferenceResponse(BaseModel):
    output: Any

@app.post("/infer", response_model=InferenceResponse)
def infer(request: InferenceRequest):
    # Dummy model: echo input
    result = request.input
    return {"output": result}

@app.get("/")
def health():
    return {"status": "ok"}

if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=8080)
