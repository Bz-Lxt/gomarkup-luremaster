"""LureMaster API smoke — Mock mode, expected cost ¥0."""
from __future__ import annotations

import math
import os
import time
import uuid

import pytest
import requests

API = os.environ.get("API_BASE", "http://127.0.0.1:29682").rstrip("/")
WEB = os.environ.get("WEB_BASE", "http://127.0.0.1:29681").rstrip("/")
SEED_EMAIL = "hunter@lure.local"
SEED_PASS = "LureHunt@2026"
MATE_EMAIL = "mate@lure.local"


def _post(path, json=None, token=None, **kw):
    h = {"Content-Type": "application/json"}
    if token:
        h["Authorization"] = f"Bearer {token}"
    return requests.post(API + path, json=json, headers=h, timeout=20, **kw)


def _get(path, token=None, **kw):
    h = {}
    if token:
        h["Authorization"] = f"Bearer {token}"
    return requests.get(API + path, headers=h, timeout=20, **kw)


def login(email=SEED_EMAIL):
    r = _post("/api/v1/auth/login", {"email": email, "password": SEED_PASS})
    assert r.status_code == 200, r.text
    body = r.json()
    assert body.get("ok") is True
    return body["data"]["access_token"]


def haversine_km(a, b, c, d):
    r = 6371.0
    p1, p2 = math.radians(a), math.radians(c)
    dp, dl = math.radians(c - a), math.radians(d - b)
    x = math.sin(dp / 2) ** 2 + math.cos(p1) * math.cos(p2) * math.sin(dl / 2) ** 2
    return 2 * r * math.asin(min(1, math.sqrt(x)))


def test_healthz():
    r = requests.get(API + "/healthz", timeout=10)
    assert r.status_code == 200
    assert r.json().get("ok") is True


def test_readyz():
    r = requests.get(API + "/readyz", timeout=10)
    assert r.status_code == 200
    assert r.json().get("ok") is True


def test_web_shell():
    r = requests.get(WEB + "/", timeout=10)
    assert r.status_code == 200
    assert "root" in r.text
    assert "index" in r.text or "Lure" in r.text or "script" in r.text


def test_seed_login_and_spots():
    token = login()
    r = _get("api/v1/spots" if False else "/api/v1/spots", token=token)
    assert r.status_code == 200
    spots = r.json()["data"]
    assert isinstance(spots, list) and len(spots) >= 3
    names = {s["name"] for s in spots}
    assert "千岛湖暗桩湾" in names
    private = next(s for s in spots if s["name"] == "千岛湖暗桩湾")
    assert private["fuzzed"] is False
    assert private["visibility"] == "PRIVATE"


def test_anti_copy_fuzz():
    hunter = login()
    spots = _get("/api/v1/spots", token=hunter).json()["data"]
    private = next(s for s in spots if s["visibility"] == "PRIVATE")
    true_lat, true_lon = private["lat"], private["lon"]
    mate = login(MATE_EMAIL)
    listed = _get("/api/v1/spots", token=mate).json()["data"]
    assert all(s["id"] != private["id"] for s in listed)
    fuzzed = _get(f"/api/v1/spots/{private['id']}", token=mate).json()["data"]
    assert fuzzed["fuzzed"] is True
    assert haversine_km(true_lat, true_lon, fuzzed["lat"], fuzzed["lon"]) >= 0.4
    again = _get(f"/api/v1/spots/{private['id']}", token=mate).json()["data"]
    assert again["lat"] == fuzzed["lat"] and again["lon"] == fuzzed["lon"]


def test_catch_hydro_bind():
    token = login()
    spots = _get("/api/v1/spots", token=token).json()["data"]
    public = next(s for s in spots if s["visibility"] == "PUBLIC" and s.get("tidal"))
    stamp = time.strftime("%Y-%m-%d %H:%M")
    r = _post(
        "/api/v1/catches",
        {
            "spot_id": public["id"],
            "local_time": stamp,
            "timezone": "Asia/Shanghai",
            "species": "YELLOWCHECK",
            "length_cm": 80.0,
            "lure_type": "MINNOW",
            "lure_color": "银白",
            "retrieve": "TWITCH",
            "layer": "SHALLOW",
            "released": True,
            "note": f"smoke-{uuid.uuid4().hex[:8]}",
        },
        token=token,
    )
    assert r.status_code in (200, 201), r.text
    cid = r.json()["data"]["id"]
    bound = None
    for _ in range(25):
        time.sleep(0.4)
        got = _get(f"/api/v1/catches/{cid}", token=token).json()["data"]
        if got.get("hydro_status") == "BOUND" or (got.get("hydro") or {}).get("status") == "BOUND":
            bound = got
            break
    assert bound is not None, "hydro not bound in time"
    hydro = bound["hydro"]
    assert "pressure_trend" in hydro
    assert "bite_score" in hydro
    assert "contributions" in hydro
    assert isinstance(hydro["contributions"], list) and len(hydro["contributions"]) >= 1


def test_recommend_cold_start():
    token = login(MATE_EMAIL)
    r = _post(
        "/api/v1/lures/recommend",
        {"species": "YELLOWCHECK", "pressure_trend": "CRASH_DOWN", "tide_window": "THIRD", "water_temp_c": 22},
        token=token,
    )
    assert r.status_code == 200, r.text
    data = r.json()["data"]
    items = data if isinstance(data, list) else data.get("items") or data.get("advice") or []
    assert len(items) >= 3


def test_slot_no_double_book():
    hunter = login()
    skipper = login("skipper@lure.local")
    acts = _get("/api/v1/activities", token=skipper).json()["data"]
    assert acts
    act = acts[0]
    open_slots = [s for s in act["slots"] if s["status"] == "OPEN"]
    assert open_slots
    sid = open_slots[0]["id"]
    aid = act["id"]
    a = _post(f"/api/v1/activities/{aid}/slots/{sid}/claim", {}, token=hunter)
    b = _post(f"/api/v1/activities/{aid}/slots/{sid}/claim", {}, token=skipper)
    winners = sum(1 for x in (a, b) if x.status_code in (200, 201) and x.json().get("ok"))
    assert winners == 1, (a.status_code, a.text, b.status_code, b.text)
    loser = a if winners and not (a.status_code in (200, 201) and a.json().get("ok")) else b
    if loser.status_code in (200, 201) and loser.json().get("ok"):
        pytest.fail("double occupy")
    assert b.status_code in (409, 200, 400, 403) or a.status_code in (409, 200, 400, 403)
