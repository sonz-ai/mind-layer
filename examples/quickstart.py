#!/usr/bin/env python3
"""Local REST example using only Python's standard library and synthetic data."""
import json
import os
import urllib.request

base = os.environ.get("MIND_LAYER_URL", "http://127.0.0.1:8080")
scope = {"agent": "assistant", "user": "synthetic-user"}


def post(path, payload):
    headers = {"Content-Type": "application/json"}
    if token := os.environ.get("MIND_LAYER_TOKEN"):
        headers["Authorization"] = "Bearer " + token
    request = urllib.request.Request(base + path, json.dumps(payload).encode(), headers)
    with urllib.request.urlopen(request, timeout=90) as response:
        return json.load(response)


memory = post("/v1/memories", {
    "scope": scope, "id": "diet", "session": "first-conversation",
    "text": "The user is vegetarian and cooks dinner for two people.",
})
hits = post("/v1/search", {"scope": scope, "query": "vegetarian dinner", "limit": 3})
assert any(hit["memory"]["id"] == memory["id"] for hit in hits)
print(json.dumps(hits, indent=2))
