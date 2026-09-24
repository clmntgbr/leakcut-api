import json
import logging
import os
import time
import uuid
from datetime import datetime, timezone
from typing import Any

import boto3
import pika
from botocore.client import Config
from engine import decode_image, env_int, load_pool, run_ocr

logger = logging.getLogger("ocr")
logging.basicConfig(level=logging.INFO)
logging.getLogger("RapidOCR").setLevel(logging.ERROR)

READY_FLAG = "/tmp/ocr-ready"


def env(name: str, default: str = "") -> str:
    return os.getenv(name, default).strip() or default


def declare_topology(channel: pika.channel.Channel) -> tuple[str, str]:
    exchange = env("RABBITMQ_EXCHANGE", "domain.events")
    queue = env("OCR_QUEUE", "ocr")
    routing_key = env("OCR_ROUTING_KEY", "video.ocr_frame_requested.v1")
    retry_ttl = int(env("RABBITMQ_RETRY_TTL_MS", "30000"))
    retry_queue = f"{queue}.retry"
    dlq = f"{queue}.dlq"
    dlx = f"{queue}.dlx"

    channel.exchange_declare(exchange=exchange, exchange_type="topic", durable=True)
    channel.exchange_declare(exchange=dlx, exchange_type="direct", durable=True)
    channel.queue_declare(
        queue=queue,
        durable=True,
        arguments={
            "x-dead-letter-exchange": dlx,
            "x-dead-letter-routing-key": "retry",
        },
    )
    try:
        channel.queue_unbind(queue=queue, exchange=exchange, routing_key="video.frames_extracted.v1")
    except Exception:
        pass
    channel.queue_bind(queue=queue, exchange=exchange, routing_key=routing_key)
    channel.queue_bind(queue=queue, exchange=dlx, routing_key="main")
    channel.queue_declare(
        queue=retry_queue,
        durable=True,
        arguments={
            "x-message-ttl": retry_ttl,
            "x-dead-letter-exchange": dlx,
            "x-dead-letter-routing-key": "main",
        },
    )
    channel.queue_bind(queue=retry_queue, exchange=dlx, routing_key="retry")
    channel.queue_declare(queue=dlq, durable=True)
    channel.queue_bind(queue=dlq, exchange=dlx, routing_key="dlq")
    return exchange, queue


def new_s3_client():
    endpoint = env("STORAGE_INTERNAL_ENDPOINT") or env("STORAGE_ENDPOINT")
    return boto3.client(
        "s3",
        endpoint_url=endpoint,
        aws_access_key_id=env("STORAGE_ACCESS_KEY"),
        aws_secret_access_key=env("STORAGE_SECRET_KEY"),
        region_name=env("STORAGE_REGION", "us-east-1"),
        config=Config(signature_version="s3v4", s3={"addressing_style": "path"}),
    )


def download_frame(s3, bucket: str, key: str) -> bytes:
    response = s3.get_object(Bucket=bucket, Key=key)
    return response["Body"].read()


def ocr_frame(s3, bucket: str, frame: dict[str, Any]) -> dict[str, Any]:
    frame_id = frame.get("id") or frame.get("frameId") or ""
    key = frame.get("storageKey") or ""
    try:
        data = download_frame(s3, bucket, key)
        text, confidence, status, lines = run_ocr(decode_image(data))
        return {
            "frameId": frame_id,
            "frameIndex": frame.get("index", 0),
            "timestampMs": frame.get("timestampMs", 0),
            "text": text,
            "confidence": confidence,
            "status": status,
            "lines": lines,
        }
    except Exception:
        logger.exception("ocr failed for frame %s", frame_id)
        return {
            "frameId": frame_id,
            "frameIndex": frame.get("index", 0),
            "timestampMs": frame.get("timestampMs", 0),
            "text": "",
            "confidence": 0.0,
            "status": "failed",
            "lines": [],
        }


def publish_batch(channel: pika.channel.Channel, exchange: str, video_id: str, results: list[dict[str, Any]]) -> None:
    event_id = str(uuid.uuid4())
    now = datetime.now(timezone.utc)
    envelope = {
        "eventId": event_id,
        "type": "video.ocr_batch_completed.v1",
        "aggregateId": video_id,
        "occurredAt": now.isoformat(),
        "payload": {
            "eventId": event_id,
            "videoId": video_id,
            "results": results,
            "timestamp": now.isoformat(),
        },
    }
    channel.basic_publish(
        exchange=exchange,
        routing_key="video.ocr_batch_completed.v1",
        body=json.dumps(envelope).encode("utf-8"),
        properties=pika.BasicProperties(
            content_type="application/json",
            delivery_mode=2,
            type="video.ocr_batch_completed.v1",
        ),
    )


def process_frame(channel: pika.channel.Channel, exchange: str, s3, payload: dict[str, Any]) -> None:
    video_id = payload.get("videoId") or ""
    frame = {
        "id": payload.get("frameId") or payload.get("id") or "",
        "index": payload.get("index", 0),
        "timestampMs": payload.get("timestampMs", 0),
        "storageKey": payload.get("storageKey") or "",
    }
    bucket = env("STORAGE_BUCKET", "media")
    logger.info(
        "ocr worker processing video=%s frameId=%s index=%s",
        video_id,
        frame["id"],
        frame["index"],
    )
    result = ocr_frame(s3, bucket, frame)
    publish_batch(channel, exchange, video_id, [result])
    logger.info(
        "ocr worker finished video=%s frameId=%s status=%s",
        video_id,
        frame["id"],
        result.get("status"),
    )


def on_message(
    channel: pika.channel.Channel,
    method: pika.spec.Basic.Deliver,
    _properties: pika.spec.BasicProperties,
    body: bytes,
    exchange: str,
    s3,
) -> None:
    try:
        envelope = json.loads(body)
        event_type = envelope.get("type") or method.routing_key
        payload = envelope.get("payload") or {}
        if isinstance(payload, str):
            payload = json.loads(payload)
        video_id = payload.get("videoId") or envelope.get("aggregateId") or ""
        frame_id = payload.get("frameId") or payload.get("id") or ""
        logger.info(
            "ocr worker received event type=%s videoId=%s frameId=%s",
            event_type,
            video_id,
            frame_id,
        )
        if event_type == "video.frames_extracted.v1" or payload.get("frames"):
            logger.info(
                "ocr worker skip batch event type=%s videoId=%s — waiting for per-frame messages",
                event_type,
                video_id,
            )
            channel.basic_ack(delivery_tag=method.delivery_tag)
            return
        if not (payload.get("storageKey") and frame_id):
            logger.warning(
                "ocr worker skip event without frameId/storageKey type=%s videoId=%s",
                event_type,
                video_id,
            )
            channel.basic_ack(delivery_tag=method.delivery_tag)
            return
        process_frame(channel, exchange, s3, payload)
        channel.basic_ack(delivery_tag=method.delivery_tag)
    except Exception:
        logger.exception("ocr worker failed, sending to retry")
        channel.basic_nack(delivery_tag=method.delivery_tag, requeue=False)


def consume() -> None:
    params = pika.URLParameters(env("RABBITMQ_URL"))
    params.heartbeat = 600
    connection = pika.BlockingConnection(params)
    channel = connection.channel()
    exchange, queue = declare_topology(channel)
    prefetch = env_int("OCR_CONCURRENCY", 1)
    channel.basic_qos(prefetch_count=prefetch)

    s3 = new_s3_client()
    with open(READY_FLAG, "w", encoding="utf-8") as handle:
        handle.write("ok")

    logger.info("ocr worker consuming queue=%s", queue)
    channel.basic_consume(
        queue=queue,
        on_message_callback=lambda ch, method, props, body: on_message(
            ch, method, props, body, exchange, s3
        ),
        auto_ack=False,
    )
    try:
        channel.start_consuming()
    except KeyboardInterrupt:
        channel.stop_consuming()
    finally:
        connection.close()


if __name__ == "__main__":
    engines = load_pool()
    logger.info("loaded %d rapidocr engine(s)", engines)
    while True:
        try:
            consume()
            break
        except pika.exceptions.AMQPConnectionError:
            logger.warning("rabbitmq unavailable, retrying in 2s")
            time.sleep(2)
