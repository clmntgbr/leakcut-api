import io
import os
import queue
from typing import Any

os.environ.setdefault("OMP_NUM_THREADS", os.getenv("OCR_INTRA_OP_THREADS", "2"))
os.environ.setdefault("ORT_INTRA_OP_NUM_THREADS", os.getenv("OCR_INTRA_OP_THREADS", "2"))
os.environ.setdefault("OPENBLAS_NUM_THREADS", "1")

import numpy as np
from PIL import Image

engine_pool: queue.Queue = queue.Queue()
engines_ready = 0


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


def load_pool() -> int:
    global engines_ready
    count = env_int("OCR_INFER_CONCURRENCY", 2)
    for _ in range(count):
        engine_pool.put(load_engine())
        engines_ready += 1
    return engines_ready


def run_ocr(image: np.ndarray) -> tuple[str, float, str]:
    engine = engine_pool.get()
    try:
        return parse_rapid(engine(image))
    finally:
        engine_pool.put(engine)


def decode_image(data: bytes) -> np.ndarray:
    image = Image.open(io.BytesIO(data)).convert("RGB")
    return np.array(image)


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
