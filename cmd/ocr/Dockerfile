FROM python:3.11-slim

WORKDIR /app

ENV FLAGS_use_mkldnn=0 \
    FLAGS_enable_pir_api=0 \
    OMP_NUM_THREADS=1 \
    MKL_NUM_THREADS=1 \
    OPENBLAS_NUM_THREADS=1 \
    KMP_AFFINITY=disabled

RUN apt-get update && apt-get install -y --no-install-recommends \
        curl \
        libgomp1 \
        libglib2.0-0 \
        libsm6 \
        libxext6 \
        libxrender1 \
        libgl1 \
    && rm -rf /var/lib/apt/lists/*

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY app.py .

# Download PP-OCR weights at build time so runtime stays offline.
RUN python -c "from app import load_engine; load_engine()"

EXPOSE 8080

CMD ["uvicorn", "app:app", "--host", "0.0.0.0", "--port", "8080", "--loop", "asyncio", "--http", "h11", "--workers", "1"]
