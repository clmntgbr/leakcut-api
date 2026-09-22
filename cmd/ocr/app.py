import os

os.environ.setdefault("OMP_NUM_THREADS", "1")
os.environ.setdefault("ORT_INTRA_OP_NUM_THREADS", "1")
os.environ.setdefault("OPENBLAS_NUM_THREADS", "1")

import base64
import io
import logging
import queue
from concurrent.futures import ThreadPoolExecutor
from typing import Any

import numpy as np
from fastapi import FastAPI, HTTPException
from PIL import Image
from pydantic import BaseModel, Field

logger = logging.getLogger("ocr")
logging.basicConfig(level=logging.INFO)

app = FastAPI(title="ocr")
engine_pool: queue.Queue = queue.Queue()
infer_pool: ThreadPoolExecutor | None = None
engines_ready = 0


class OCRImage(BaseModel):
    frame_id: str
    image_base64: str


class OCRRequest(BaseModel):
    images: list[OCRImage] = Field(default_factory=list)
    lang: str = "fr+en"


class OCRItem(BaseModel):
    frame_id: str
    text: str
    confidence: float
    status: str


class OCRResponse(BaseModel):
    results: list[OCRItem]


def env_int(name: str, default: int) -> int:
    raw = os.getenv(name, "")
    if not raw:
        return default
    try:
        value = int(raw)
    except ValueError:
        return default
    return max(1, value)


def load_engine():
    from rapidocr import RapidOCR

    return RapidOCR()


@app.on_event("startup")
def startup() -> None:
    global infer_pool, engines_ready
    infer_concurrency = env_int("OCR_INFER_CONCURRENCY", 2)
    logger.info("loading %d rapidocr engine(s)", infer_concurrency)
    for index in range(infer_concurrency):
        engine_pool.put(load_engine())
        engines_ready += 1
        logger.info("ocr engine %d/%d ready", index + 1, infer_concurrency)
    infer_pool = ThreadPoolExecutor(max_workers=infer_concurrency, thread_name_prefix="ocr-infer")


@app.on_event("shutdown")
def shutdown() -> None:
    if infer_pool is not None:
        infer_pool.shutdown(wait=False, cancel_futures=True)


@app.get("/health")
def health() -> dict[str, Any]:
    if engines_ready == 0:
        raise HTTPException(status_code=503, detail="model not loaded")
    return {"status": "ok", "engine": "rapidocr", "engines": engines_ready}


@app.post("/ocr", response_model=OCRResponse)
def recognize(payload: OCRRequest) -> OCRResponse:
    if engines_ready == 0:
        raise HTTPException(status_code=503, detail="model not loaded")
    if infer_pool is None or len(payload.images) <= 1:
        return OCRResponse(results=[process_image(image) for image in payload.images])

    futures = [infer_pool.submit(process_image, image) for image in payload.images]
    return OCRResponse(results=[future.result() for future in futures])


def process_image(image: OCRImage) -> OCRItem:
    try:
        array = decode_image(image.image_base64)
        text, confidence, status = run_ocr(array)
        return OCRItem(
            frame_id=image.frame_id,
            text=text,
            confidence=confidence,
            status=status,
        )
    except Exception as exc:  # isolated frame failure
        logger.exception("ocr failed for frame %s: %s", image.frame_id, exc)
        return OCRItem(
            frame_id=image.frame_id,
            text="",
            confidence=0.0,
            status="failed",
        )


def decode_image(raw_b64: str) -> np.ndarray:
    data = base64.b64decode(raw_b64)
    image = Image.open(io.BytesIO(data)).convert("RGB")
    return np.array(image)


def run_ocr(image: np.ndarray) -> tuple[str, float, str]:
    engine = engine_pool.get()
    try:
        return parse_rapid(engine(image))
    finally:
        engine_pool.put(engine)


def parse_rapid(raw: Any) -> tuple[str, float, str]:
    if raw is None:
        return "", 0.0, "empty"

    texts = getattr(raw, "txts", None)
    scores = getattr(raw, "scores", None)
    if texts is not None:
        return join_lines(_as_list(texts), _as_float_list(scores))

    if isinstance(raw, tuple):
        raw = raw[0] if raw else None
    if not raw:
        return "", 0.0, "empty"

    lines: list[str] = []
    confidences: list[float] = []
    for item in raw:
        if not isinstance(item, (list, tuple)) or len(item) < 2:
            continue
        if len(item) >= 3:
            text = str(item[1]).strip()
            score = _as_float(item[2])
        else:
            text = str(item[0]).strip()
            score = _as_float(item[1])
        if text:
            lines.append(text)
            if score is not None:
                confidences.append(score)
    return join_lines(lines, confidences)


def join_lines(lines: list[Any], scores: list[float]) -> tuple[str, float, str]:
    cleaned = [str(line).strip() for line in lines if str(line).strip()]
    if not cleaned:
        return "", 0.0, "empty"
    confidence = sum(scores) / len(scores) if scores else 0.0
    return "\n".join(cleaned), confidence, "success"


def _as_list(value: Any) -> list[Any]:
    if value is None:
        return []
    if isinstance(value, str):
        return [value]
    if isinstance(value, (list, tuple)):
        return list(value)
    return [value]


def _as_float(value: Any) -> float | None:
    try:
        return float(value)
    except (TypeError, ValueError):
        return None


def _as_float_list(value: Any) -> list[float]:
    out: list[float] = []
    for item in _as_list(value):
        parsed = _as_float(item)
        if parsed is not None:
            out.append(parsed)
    return out
