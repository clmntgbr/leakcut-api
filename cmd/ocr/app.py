import os

os.environ.setdefault("FLAGS_use_mkldnn", "0")
os.environ.setdefault("FLAGS_enable_pir_api", "0")
os.environ.setdefault("OMP_NUM_THREADS", "1")
os.environ.setdefault("MKL_NUM_THREADS", "1")
os.environ.setdefault("OPENBLAS_NUM_THREADS", "1")
os.environ.setdefault("KMP_AFFINITY", "disabled")

import base64
import io
import logging
import threading
from typing import Any

import numpy as np
from fastapi import FastAPI, HTTPException
from PIL import Image
from pydantic import BaseModel, Field

logger = logging.getLogger("ocr")
logging.basicConfig(level=logging.INFO)

app = FastAPI(title="ocr")
ocr_engine = None
ocr_lock = threading.Lock()


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


def normalize_lang(lang: str) -> str:
    value = (lang or "en").lower()
    if "+" in value or ("fr" in value and "en" in value):
        return "latin"
    if "fr" in value:
        return "french"
    if "en" in value:
        return "en"
    return value.split("+")[0]


def load_engine():
    from paddleocr import PaddleOCR

    lang = normalize_lang(os.getenv("OCR_LANG", "fr+en"))
    preferred = os.getenv("PADDLEOCR_VERSION", "PP-OCRv5")
    for version in (preferred, "PP-OCRv5"):
        try:
            return PaddleOCR(
                use_doc_orientation_classify=False,
                use_doc_unwarping=False,
                use_textline_orientation=False,
                lang=lang,
                ocr_version=version,
            )
        except TypeError:
            try:
                return PaddleOCR(lang=lang, ocr_version=version)
            except Exception as exc:
                logger.warning("failed to init paddleocr %s: %s", version, exc)
        except Exception as exc:
            logger.warning("failed to init paddleocr %s: %s", version, exc)
    return PaddleOCR(lang=lang)


@app.on_event("startup")
def startup() -> None:
    global ocr_engine
    logger.info("loading ocr models")
    ocr_engine = load_engine()
    logger.info("ocr models ready")


@app.get("/health")
def health() -> dict[str, str]:
    if ocr_engine is None:
        raise HTTPException(status_code=503, detail="model not loaded")
    return {"status": "ok"}


@app.post("/ocr", response_model=OCRResponse)
def recognize(payload: OCRRequest) -> OCRResponse:
    if ocr_engine is None:
        raise HTTPException(status_code=503, detail="model not loaded")

    results: list[OCRItem] = []
    for image in payload.images:
        try:
            array = decode_image(image.image_base64)
            text, confidence, status = run_ocr(array)
            results.append(
                OCRItem(
                    frame_id=image.frame_id,
                    text=text,
                    confidence=confidence,
                    status=status,
                )
            )
        except Exception as exc:  # isolated frame failure
            logger.exception("ocr failed for frame %s: %s", image.frame_id, exc)
            results.append(
                OCRItem(
                    frame_id=image.frame_id,
                    text="",
                    confidence=0.0,
                    status="failed",
                )
            )
    return OCRResponse(results=results)


def decode_image(raw_b64: str) -> np.ndarray:
    data = base64.b64decode(raw_b64)
    image = Image.open(io.BytesIO(data)).convert("RGB")
    return np.array(image)


def run_ocr(image: np.ndarray) -> tuple[str, float, str]:
    with ocr_lock:
        if hasattr(ocr_engine, "predict"):
            raw = ocr_engine.predict(image)
            return parse_predict(raw)
        raw = ocr_engine.ocr(image)
        return parse_legacy(raw)


def parse_predict(raw: Any) -> tuple[str, float, str]:
    if not raw:
        return "", 0.0, "empty"

    first = raw[0] if isinstance(raw, list) else raw
    texts = _attr_or_key(first, "rec_texts")
    scores = _attr_or_key(first, "rec_scores")
    if texts is None and isinstance(first, dict):
        texts = first.get("rec_text") or first.get("text")
        scores = first.get("rec_score") or first.get("confidence")

    lines = _as_list(texts)
    confidences = _as_float_list(scores)
    if not lines:
        return "", 0.0, "empty"

    cleaned = [str(line).strip() for line in lines if str(line).strip()]
    if not cleaned:
        return "", 0.0, "empty"

    confidence = sum(confidences) / len(confidences) if confidences else 0.0
    return "\n".join(cleaned), confidence, "success"


def parse_legacy(raw: Any) -> tuple[str, float, str]:
    if not raw:
        return "", 0.0, "empty"

    pages = raw if isinstance(raw, list) else [raw]
    lines: list[str] = []
    scores: list[float] = []
    for page in pages:
        if not page:
            continue
        for item in page:
            if not item or len(item) < 2:
                continue
            rec = item[1]
            if isinstance(rec, (list, tuple)) and rec:
                text = str(rec[0]).strip()
                score = float(rec[1]) if len(rec) > 1 else 0.0
            else:
                text = str(rec).strip()
                score = 0.0
            if text:
                lines.append(text)
                scores.append(score)

    if not lines:
        return "", 0.0, "empty"
    confidence = sum(scores) / len(scores) if scores else 0.0
    return "\n".join(lines), confidence, "success"


def _attr_or_key(obj: Any, name: str) -> Any:
    if obj is None:
        return None
    if hasattr(obj, name):
        return getattr(obj, name)
    if isinstance(obj, dict):
        return obj.get(name)
    return None


def _as_list(value: Any) -> list[Any]:
    if value is None:
        return []
    if isinstance(value, str):
        return [value]
    if isinstance(value, (list, tuple)):
        return list(value)
    return [value]


def _as_float_list(value: Any) -> list[float]:
    out: list[float] = []
    for item in _as_list(value):
        try:
            out.append(float(item))
        except (TypeError, ValueError):
            continue
    return out
