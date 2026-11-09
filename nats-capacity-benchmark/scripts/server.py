#!/usr/bin/env python3
"""
Lightweight zero-dependency runner server for NATS Capacity Benchmark.
Serves static/report.html and provides API endpoints to trigger benchmark runs.
"""

import http.server
import json
import os
import socketserver
import subprocess
import sys
import threading
from pathlib import Path

ROOT_DIR = Path(__file__).resolve().parent.parent
PORT = int(os.environ.get("BENCHMARK_PORT", "8080"))

status_lock = threading.Lock()
current_status = {
    "running": False,
    "test_id": "",
    "exit_code": None,
    "recent_logs": []
}

def run_benchmark_task(env_vars):
    global current_status
    test_id = env_vars.get("TEST_ID", "web-run")
    cmd = ["bash", str(ROOT_DIR / "benchmark.sh")]

    with status_lock:
        current_status["running"] = True
        current_status["test_id"] = test_id
        current_status["exit_code"] = None
        current_status["recent_logs"] = [f"Starting benchmark test: {test_id}"]

    run_env = os.environ.copy()
    run_env.update(env_vars)

    proc = subprocess.Popen(
        cmd,
        cwd=str(ROOT_DIR),
        env=run_env,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        text=True
    )

    for line in iter(proc.stdout.readline, ''):
        line = line.rstrip()
        if line:
            with status_lock:
                current_status["recent_logs"].append(line)
                if len(current_status["recent_logs"]) > 200:
                    current_status["recent_logs"].pop(0)

    proc.wait()
    with status_lock:
        current_status["running"] = False
        current_status["exit_code"] = proc.returncode
        current_status["recent_logs"].append(f"Completed with exit code {proc.returncode}")

class BenchmarkRequestHandler(http.server.SimpleHTTPRequestHandler):
    def __init__(self, *args, **kwargs):
        super().__init__(*args, directory=str(ROOT_DIR / "static"), **kwargs)

    def do_GET(self):
        if self.path == "/api/status":
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.send_header("Access-Control-Allow-Origin", "*")
            self.end_headers()
            with status_lock:
                data = json.dumps(current_status)
            self.wfile.write(data.encode("utf-8"))
        elif self.path == "/" or self.path == "/index.html":
            self.path = "/report.html"
            return super().do_GET()
        else:
            return super().do_GET()

    def do_POST(self):
        if self.path == "/api/run":
            content_len = int(self.headers.get("Content-Length", 0))
            body = self.rfile.read(content_len)
            params = {}
            if body:
                try:
                    params = json.loads(body.decode("utf-8"))
                except Exception:
                    params = {}

            with status_lock:
                if current_status["running"]:
                    self.send_response(409)
                    self.send_header("Content-Type", "application/json")
                    self.end_headers()
                    self.wfile.write(b'{"error": "A benchmark is already running"}')
                    return

            env_vars = {}
            if "scenario" in params: env_vars["SCENARIO"] = str(params["scenario"])
            if "throughput" in params: env_vars["THROUGHPUT"] = str(params["throughput"])
            if "clients" in params: env_vars["CLIENTS"] = str(params["clients"])
            if "duration" in params: env_vars["DURATION"] = str(params["duration"])
            if "size" in params: env_vars["SIZE"] = str(params["size"])
            if "subject" in params: env_vars["SUBJECT"] = str(params["subject"])
            if "test_id" in params: env_vars["TEST_ID"] = str(params["test_id"])

            t = threading.Thread(target=run_benchmark_task, args=(env_vars,), daemon=True)
            t.start()

            self.send_response(202)
            self.send_header("Content-Type", "application/json")
            self.send_header("Access-Control-Allow-Origin", "*")
            self.end_headers()
            resp = json.dumps({"status": "started", "test_id": env_vars.get("TEST_ID", "web-run")})
            self.wfile.write(resp.encode("utf-8"))
        else:
            self.send_response(404)
            self.end_headers()

def main():
    socketserver.TCPServer.allow_reuse_address = True
    with socketserver.TCPServer(("", PORT), BenchmarkRequestHandler) as httpd:
        print(f"Benchmark UI Server running at: http://localhost:{PORT}")
        print("Press Ctrl+C to terminate.")
        try:
            httpd.serve_forever()
        except KeyboardInterrupt:
            print("\nShutting down server.")

if __name__ == "__main__":
    main()

