# ── BERT FastAPI classifier ──────────────────────────────────────────────────
# Serves the fine-tuned BERTimbau model that classifies patent abstracts
# into 8 IPC sections (A..H).
#
# NOTE: image is heavy (~2 GB) because torch + transformers + model weights.
# For production, prefer Hugging Face Spaces or a dedicated ML host.

FROM python:3.11-slim

WORKDIR /app

# System deps for transformers
RUN apt-get update && apt-get install -y --no-install-recommends \
      curl build-essential && \
    rm -rf /var/lib/apt/lists/*

# Cache the heavy pip install layer
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# App code + model (model is mounted via volume in docker-compose for size)
COPY api_argos.py ./

EXPOSE 8000

# Health check used by docker-compose to know AI is ready.
HEALTHCHECK --interval=10s --timeout=5s --start-period=30s \
  CMD curl -fsS http://localhost:8000/health || exit 1

CMD ["uvicorn", "api_argos:app", "--host", "0.0.0.0", "--port", "8000"]
