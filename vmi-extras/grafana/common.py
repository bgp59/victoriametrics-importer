#! /usr/bin/env python3


import re

default_grafana_root_url = "http://localhost:3000"
default_grafana_user = "admin"
default_grafana_password = "vmi"
default_grafana_folder = "General"
vmi_folder = "vmi-reference"
ref_dashboard_suffix = "-ref"
wip_dashboard_suffix = " (WIP)"
default_out_subdir = "dashboards"


def normalize_title(title: str) -> str:
    # camelCaseTitle -> camel-case-title
    normal_title = re.sub(r"([a-z])([A-Z])", r"\1-\2", title).lower()
    # non-standard chars -> -
    normal_title = re.sub(r"[^a-z_0-9]+", "-", normal_title)
    return normal_title


def wip_to_ref_title(title: str) -> str:
    if title.lower().endswith(wip_dashboard_suffix.lower()):
        title = title[: -len(wip_dashboard_suffix)]
    if not title.lower().endswith(ref_dashboard_suffix.lower()):
        title += ref_dashboard_suffix
    return title


def ref_to_wip_title(title: str) -> str:
    if not title.lower().endswith(wip_dashboard_suffix.lower()):
        title += wip_dashboard_suffix
    return title
