# Grafana HTTP API Python Client

import re
import urllib.parse
from typing import List, Optional

import requests


class GrafanaClient:
    def __init__(
        self,
        url: str = "http://localhost:3000",
        user: Optional[str] = None,
        password: Optional[str] = None,
    ):
        self._url = url
        self._auth = (user, password) if user and password else None

    def search_folder(self, title: str) -> List[dict]:
        r = requests.get(
            f"{self._url}/api/search?query={urllib.parse.quote(title)}&type=dash-folder",
            auth=self._auth,
        )
        r.raise_for_status()
        return [meta for meta in r.json() if meta["title"] == title]

    def search_dashboard(self, title: str) -> List[dict]:
        r = requests.get(
            f"{self._url}/api/search?query={urllib.parse.quote(title)}&type=dash-db",
            auth=self._auth,
        )
        r.raise_for_status()
        return [meta for meta in r.json() if meta["title"] == title]

    def create_folder(self, title: str, uid: Optional[str] = None):
        post_data = {
            "title": title,
        }
        if uid:
            post_data["uid"] = uid
        r = requests.post(f"{self._url}/api/folders", json=post_data, auth=self._auth)
        r.raise_for_status()
        return r.json()

    def get_dashboard(self, uid: str) -> dict:
        r = requests.get(f"{self._url}/api/dashboards/uid/{uid}", auth=self._auth)
        r.raise_for_status()
        return r.json()

    def save_dashboard(
        self,
        dashboard: dict,
        folder_uid: Optional[str] = None,
        message: Optional[str] = None,
    ) -> dict:
        post_data = {
            "dashboard": dashboard,
        }
        if folder_uid:
            post_data["folderUid"] = folder_uid
        if message:
            post_data["message"] = message
        r = requests.post(
            f"{self._url}/api/dashboards/db", json=post_data, auth=self._auth
        )
        r.raise_for_status()
        return r.json()


def normalize_title(title: str) -> str:
    # camelCaseTitle -> camel-case-title
    normal_title = re.sub(r"([a-z])([A-Z])", r"\1-\2", title).lower()
    # non-standard chars -> -:
    normal_title = re.sub(r"[^a-z_0-9]+", "-", normal_title)
    # Drop start/end "-" if any:
    normal_title = re.sub(r"^-+|-+$", "", normal_title)
    return normal_title


def from_to_suffix_title(title: str, from_suffix: str, to_suffix: str) -> str:
    if title.lower().endswith(from_suffix.lower()):
        title = title[: -len(from_suffix)]
    if not title.lower().endswith(to_suffix.lower()):
        title += to_suffix
    return title


def folder_dash_title(folder: str, dash: str) -> str:
    return f"{folder}/{dash}" if folder else dash
