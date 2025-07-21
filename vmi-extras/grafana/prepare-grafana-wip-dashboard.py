#! /usr/bin/env python3

"""
All VMI dashboards are provisioned so they cannot be changed in place. This
script will make an editable copy under the General folder.
"""

import argparse
import sys
import urllib.parse

import requests

from common import (
    default_grafana_folder,
    default_grafana_password,
    default_grafana_root_url,
    default_grafana_user,
    wip_dashboard_suffix,
)

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "-r",
        "--root-url",
        default=default_grafana_root_url,
        help="""Grafana root URL, default: %(default)r.""",
    )
    parser.add_argument(
        "-u",
        "--user",
        default=default_grafana_user,
        help="""Grafana user, default: %(default)r.""",
    )
    parser.add_argument(
        "-p",
        "--password",
        default=default_grafana_password,
        help="""Grafana password, default: %(default)r.""",
    )
    parser.add_argument(
        "dashboard_title",
        metavar="DASHBOARD_TITLE",
    )
    args = parser.parse_args()
    root_url = args.root_url
    auth = (args.user, args.password)
    ref_dashboard_title = args.dashboard_title
    r = requests.get(
        f"{root_url}/api/search?query={urllib.parse.quote(ref_dashboard_title)}",
        auth=auth,
    )
    r.raise_for_status()
    ref_uid, ref_folder = None, default_grafana_folder
    for meta in r.json():
        if meta["title"] == ref_dashboard_title:
            ref_folder = meta["folderTitle"]
            ref_uid = meta["uid"]
            break
    if ref_uid is None:
        raise RuntimeError(f"{ref_dashboard_title!r}: No such dashboard")
    r = requests.get(f"{root_url}/api/dashboards/uid/{ref_uid}", auth=auth)
    r.raise_for_status()
    dashboard_meta = r.json()
    dashboard, meta = dashboard_meta["dashboard"], dashboard_meta["meta"]
    dashboard["title"] += wip_dashboard_suffix
    dashboard["id"] = None
    dashboard["uid"] = None
    post_data = {
        "dashboard": dashboard,
        "message": f"From {ref_dashboard_title} under {ref_folder}",
        "overwrite": True,
    }
    r = requests.post(f"{root_url}/api/dashboards/db", json=post_data, auth=auth)
    r.raise_for_status()
    print(
        f'Created "{default_grafana_folder}/{dashboard.get("title")}" from "{ref_folder}/{ref_dashboard_title}"',
        file=sys.stderr,
    )
