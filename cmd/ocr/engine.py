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


def run_ocr(image: np.ndarray) -> tuple[str, float, str, list[dict[str, Any]]]:
    engine = engine_pool.get()
    try:
        return parse_rapid(engine(image))
    finally:
        engine_pool.put(engine)


def decode_image(data: bytes) -> np.ndarray:
    image = Image.open(io.BytesIO(data)).convert("RGB")
    return np.array(image)


def parse_rapid(raw: Any) -> tuple[str, float, str, list[dict[str, Any]]]:
    if raw is None:
        return "", 0.0, "empty", []

    texts = getattr(raw, "txts", None)
    scores = getattr(raw, "scores", None)
    boxes = getattr(raw, "boxes", None)
    if texts is not None:
        return summarize_lines(zip_lines(_as_list(texts), _as_float_list(scores), _as_boxes(boxes)))

    if isinstance(raw, tuple):
        raw = raw[0] if raw else None
    if not raw:
        return "", 0.0, "empty", []

    lines: list[dict[str, Any]] = []
    for item in raw:
        if not isinstance(item, (list, tuple)) or len(item) < 2:
            continue
        if len(item) >= 3:
            box = _as_box(item[0])
            text = str(item[1]).strip()
            score = _as_float(item[2])
        else:
            box = []
            text = str(item[0]).strip()
            score = _as_float(item[1])
        if text:
            lines.append({
                "text": text,
                "confidence": score if score is not None else 0.0,
                "box": box,
            })
    return summarize_lines(lines)


def zip_lines(texts: list[Any], scores: list[float], boxes: list[list[dict[str, int]]]) -> list[dict[str, Any]]:
    lines: list[dict[str, Any]] = []
    for index, text in enumerate(texts):
        cleaned = str(text).strip()
        if not cleaned:
            continue
        score = scores[index] if index < len(scores) else 0.0
        box = boxes[index] if index < len(boxes) else []
        lines.append({"text": cleaned, "confidence": score, "box": box})
    return lines


def summarize_lines(lines: list[dict[str, Any]]) -> tuple[str, float, str, list[dict[str, Any]]]:
    if not lines:
        return "", 0.0, "empty", []
    texts = [str(line["text"]) for line in lines]
    scores = [float(line["confidence"]) for line in lines]
    confidence = sum(scores) / len(scores) if scores else 0.0
    return "\n".join(texts), confidence, "success", lines


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


def _as_boxes(value: Any) -> list[list[dict[str, int]]]:
    if value is None:
        return []
    if hasattr(value, "tolist"):
        value = value.tolist()
    if not isinstance(value, (list, tuple)):
        return []
    return [_as_box(item) for item in value]


def _as_box(value: Any) -> list[dict[str, int]]:
    if value is None:
        return []
    if hasattr(value, "tolist"):
        value = value.tolist()
    if not isinstance(value, (list, tuple)):
        return []
    if len(value) == 8 and all(isinstance(item, (int, float)) for item in value):
        value = [[value[i], value[i + 1]] for i in range(0, 8, 2)]
    points: list[dict[str, int]] = []
    for point in value:
        if hasattr(point, "tolist"):
            point = point.tolist()
        if not isinstance(point, (list, tuple)) or len(point) < 2:
            continue
        x = _as_float(point[0])
        y = _as_float(point[1])
        if x is None or y is None:
            continue
        points.append({"x": int(round(x)), "y": int(round(y))})
    if len(points) != 4:
        return []
    return points
