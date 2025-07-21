#! /usr/bin/env python3

"""
Save a WIP VMI dashboard into the provisioned area.
"""

import argparse
import json
import os
import sys
import time
import urllib.parse

import requests

from common import (
    default_grafana_folder,
    default_grafana_password,
    default_grafana_root_url,
    default_grafana_user,
    default_out_subdir,
    vmi_folder,
    normalize_title,
    ref_dashboard_suffix,
    ref_to_wip_title,
    wip_dashboard_suffix,
    wip_to_ref_title,
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
        "-f",
        "--folder",
        default=vmi_folder,
        help="""Grafana folder, default: %(default)r.""",
    )
    parser.add_argument(
        "-k",
        "--keep",
        default=False,
        help="""Keep Instance and Hostname var selection. By default they are
             either cleared or set to All if the latter is enabled.""",
    )
    parser.add_argument(
        "-t",
        "--title",
        help=f"""
        New title, if not provided it will be inferred from WIP_DASHBOARD_TITLE
        with {wip_dashboard_suffix!r} suffix removed and
        {ref_dashboard_suffix!r} suffix appended as needed.
        """,
    )
    parser.add_argument(
        "-o",
        "--out-dir",
        help=f"""
        Output dir, default is the location of this script/{default_out_subdir}
        """,
    )
    parser.add_argument(
        "dashboard_title",
        metavar="DASHBOARD_TITLE",
        help=f"""
        The reference or WIP title. The {wip_dashboard_suffix!r} suffix will be
        appended as needed.
        """,
    )
    args = parser.parse_args()
    root_url = args.root_url
    auth = (args.user, args.password)
    title = args.title
    folder = args.folder
    out_dir = args.out_dir
    if out_dir is None:
        out_dir = os.path.join(
            os.path.dirname(os.path.abspath(__file__)),
            default_out_subdir,
        )
    wip_dashboard_title = ref_to_wip_title(args.dashboard_title)
    r = requests.get(
        f"{root_url}/api/search?query={urllib.parse.quote(wip_dashboard_title)}",
        auth=auth,
    )
    r.raise_for_status()
    wip_uid, wip_folder = None, default_grafana_folder
    for meta in r.json():
        if meta["title"] == wip_dashboard_title:
            wip_uid, wip_folder = (
                meta["uid"],
                meta.get("folderTitle", default_grafana_folder),
            )
            break
    if wip_uid is None:
        raise RuntimeError(f"{wip_dashboard_title!r}: No such dashboard")
    r = requests.get(f"{root_url}/api/dashboards/uid/{wip_uid}", auth=auth)
    r.raise_for_status()
    dashboard_meta = r.json()
    dashboard = dashboard_meta["dashboard"]
    if title is None:
        title = wip_to_ref_title(dashboard["title"])
    norm_title = normalize_title(title)
    dash_out_dir = os.path.join(out_dir, folder)
    out_file = os.path.join(dash_out_dir, f"{norm_title}.json")
    dashboard["id"] = None
    dashboard["uid"] = norm_title
    dashboard["title"] = title
    dashboard["version"] = int(time.time())
    if not args.keep and "templating" in dashboard:
        templates = dashboard["templating"].get("list", [])
        for template in templates:
            if template.get("name", "").lower() not in {"instance", "hostname"}:
                continue
            if template.get("includeAll"):
                template["allValue"] = ".*"
                if template["multi"]:
                    current = {
                        "selected": True,
                        "text": ["All"],
                        "value": ["$__all"],
                    }
                else:
                    current = {
                        "selected": True,
                        "text": "All",
                        "value": "$__all",
                    }
            else:
                current = {
                    "selected": False,
                    "text": "",
                    "value": "",
                }
            template["current"] = current
    os.makedirs(dash_out_dir, exist_ok=True)
    with open(out_file, "wt") as f:
        json.dump(dashboard, f, indent=2)
        f.write("\n")
    print(
        f'Dashboard "{wip_folder}/{wip_dashboard_title}" saved into "{out_file}" using "{dashboard["title"]}" title',
        file=sys.stderr,
    )
