#!/usr/bin/env python3
import json
import random
import time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from urllib.parse import urlparse

BASE_DIR = Path(__file__).resolve().parent
DATA_DIR = BASE_DIR / "data"
API_KEYS_PATH = DATA_DIR / "api_keys.json"
USAGE_LOG_PATH = DATA_DIR / "usage.log"

VIDEO_SOURCES = [
    "http://api.xingchenfu.xyz/API/hssp.php",
    "http://api.xingchenfu.xyz/API/wmsc.php",
    "http://api.xingchenfu.xyz/API/tianmei.php",
    "http://api.xingchenfu.xyz/API/cdxl.php",
    "http://api.xingchenfu.xyz/API/yzxl.php",
    "http://api.xingchenfu.xyz/API/rwsp.php",
    "http://api.xingchenfu.xyz/API/nvda.php",
    "http://api.xingchenfu.xyz/API/bsxl.php",
    "http://api.xingchenfu.xyz/API/zzxjj.php",
    "http://api.xingchenfu.xyz/API/qttj.php",
    "http://api.xingchenfu.xyz/API/xqtj.php",
    "http://api.xingchenfu.xyz/API/sktj.php",
    "http://api.xingchenfu.xyz/API/cossp.php",
    "http://api.xingchenfu.xyz/API/xiaohulu.php",
    "http://api.xingchenfu.xyz/API/manhuay.php",
    "http://api.xingchenfu.xyz/API/bianzhuang.php",
    "http://api.xingchenfu.xyz/API/jk.php",
    "https://v2.xxapi.cn/api/meinv?return=302",
    "https://api.jkyai.top/API/jxhssp.php",
    "https://api.jkyai.top/API/jxbssp.php",
    "https://api.jkyai.top/API/rmtmsp/api.php",
    "https://api.jkyai.top/API/qcndxl.php",
    "https://www.hhlqilongzhu.cn/api/MP4_xiaojiejie.php",
]

RATE_LIMIT_WINDOW_SECONDS = 60


def load_api_keys():
    if not API_KEYS_PATH.exists():
        return {}
    with API_KEYS_PATH.open("r", encoding="utf-8") as handle:
        payload = json.load(handle)
    return {key["key"]: key for key in payload.get("keys", [])}


def record_usage(api_key, endpoint, status_code):
    timestamp = time.strftime("%Y-%m-%d %H:%M:%S", time.localtime())
    DATA_DIR.mkdir(exist_ok=True)
    with USAGE_LOG_PATH.open("a", encoding="utf-8") as handle:
        handle.write(f"{timestamp}\t{api_key}\t{endpoint}\t{status_code}\n")


def parse_api_key(headers):
    auth = headers.get("Authorization")
    if auth and auth.lower().startswith("bearer "):
        return auth.split(" ", 1)[1].strip()
    api_key = headers.get("X-API-Key")
    if api_key:
        return api_key.strip()
    return None


class RateLimiter:
    def __init__(self):
        self.request_log = {}

    def allow(self, api_key, limit):
        now = time.time()
        window_start = now - RATE_LIMIT_WINDOW_SECONDS
        entries = self.request_log.setdefault(api_key, [])
        entries[:] = [ts for ts in entries if ts >= window_start]
        if len(entries) >= limit:
            return False
        entries.append(now)
        return True


class ApiHandler(BaseHTTPRequestHandler):
    rate_limiter = RateLimiter()

    def _send_json(self, status_code, payload):
        response = json.dumps(payload, ensure_ascii=False).encode("utf-8")
        self.send_response(status_code)
        self.send_header("Content-Type", "application/json; charset=utf-8")
        self.send_header("Content-Length", str(len(response)))
        self.end_headers()
        self.wfile.write(response)

    def _unauthorized(self, message="Missing or invalid API key"):
        self._send_json(401, {"error": message})

    def _rate_limited(self, limit):
        self._send_json(429, {"error": "Rate limit exceeded", "limit_per_minute": limit})

    def _not_found(self):
        self._send_json(404, {"error": "Not found"})

    def do_GET(self):
        parsed = urlparse(self.path)
        if parsed.path == "/health":
            self._send_json(200, {"status": "ok", "timestamp": int(time.time())})
            return

        if parsed.path != "/v1/video/random":
            self._not_found()
            return

        api_key = parse_api_key(self.headers)
        keys = load_api_keys()
        if not api_key or api_key not in keys:
            record_usage(api_key or "unknown", parsed.path, 401)
            self._unauthorized()
            return

        plan = keys[api_key].get("plan", "basic")
        limit = keys[api_key].get("rate_limit_per_minute", 30)
        if not self.rate_limiter.allow(api_key, limit):
            record_usage(api_key, parsed.path, 429)
            self._rate_limited(limit)
            return

        url = random.choice(VIDEO_SOURCES)
        payload = {
            "url": url,
            "plan": plan,
            "cache_ttl": 0,
            "timestamp": int(time.time()),
        }
        record_usage(api_key, parsed.path, 200)
        self._send_json(200, payload)

    def log_message(self, format, *args):
        return


def main():
    DATA_DIR.mkdir(exist_ok=True)
    server = ThreadingHTTPServer(("0.0.0.0", 8080), ApiHandler)
    print("API server running on http://0.0.0.0:8080")
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        pass
    finally:
        server.server_close()


if __name__ == "__main__":
    main()
