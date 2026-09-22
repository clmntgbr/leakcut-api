FROM python:3.11-slim

WORKDIR /app

ENV OMP_NUM_THREADS=1 \
    ORT_INTRA_OP_NUM_THREADS=1 \
    OPENBLAS_NUM_THREADS=1 \
    RAPIDOCR_HOME=/root/.rapidocr

RUN apt-get update && apt-get install -y --no-install-recommends \
        curl \
        libgomp1 \
        libglib2.0-0 \
        libgl1 \
    && rm -rf /var/lib/apt/lists/*

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY app.py .

# Download RapidOCR ONNX weights at build time so runtime stays offline.
RUN python -c "from app import load_engine; load_engine()"

EXPOSE 8080

CMD ["uvicorn", "app:app", "--host", "0.0.0.0", "--port", "8080", "--loop", "asyncio", "--http", "h11", "--workers", "1"]
