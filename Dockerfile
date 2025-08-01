# Python-based Dockerfile for FastAPI model serving
FROM python:3.11-slim

WORKDIR /app

# Install dependencies
COPY python/requirements.txt ./requirements.txt
RUN pip install --no-cache-dir -r requirements.txt

# Copy model server code
COPY python/kubernetes_mcp_server/ ./kubernetes_mcp_server/

EXPOSE 8080

ENTRYPOINT ["uvicorn", "kubernetes_mcp_server.inference_server:app", "--host", "0.0.0.0", "--port", "8080"]