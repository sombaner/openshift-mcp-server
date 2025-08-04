from fastapi import FastAPI, Request
from pydantic import BaseModel
from typing import Any
import uvicorn

app = FastAPI()

class InferenceResponse(BaseModel):
    output: Any

@app.post("/infer", response_model=InferenceResponse)
async def infer(request: Request):
    data = await request.json()
    # Accept both 'input' and 'inputs' keys
    if "input" in data:
        result = data["input"]
    elif "inputs" in data:
        result = data["inputs"]
    else:
        return {"output": None}
    return {"output": result}

@app.get("/")
def health():
    return {"status": "ok"}

if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=8080)
