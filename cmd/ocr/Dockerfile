FROM python:3.11-slim

WORKDIR /app

ENV OMP_NUM_THREADS=1 \
    ORT_INTRA_OP_NUM_THREADS=1 \
    OPENBLAS_NUM_THREADS=1 \
    RAPIDOCR_HOME=/root/.rapidocr

RUN apt-get update && apt-get install -y --no-install-recommends \
        libgomp1 \
        libglib2.0-0 \
        libgl1 \
    && rm -rf /var/lib/apt/lists/*

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY engine.py worker.py .

RUN python -c "from engine import load_engine; load_engine()"

CMD ["python", "worker.py"]
