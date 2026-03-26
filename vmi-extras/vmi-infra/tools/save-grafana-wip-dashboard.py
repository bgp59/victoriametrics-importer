#! /usr/bin/env python3

description = """
Save work-in-progress dashboard into the provisioned area.
"""

import argparse
import json
import os
import sys
import time

from definitions import (
    default_folder,
    default_grafana_password,
    default_grafana_root_url,
    default_grafana_user,
    default_out_subdir,
    ref_dashboard_title_suffix,
    wip_dashboard_title_suffix,
)
from grafana import (
    GrafanaClient,
    folder_dash_title,
    from_to_suffix_title,
    normalize_title,
)
from help_formatter import CustomWidthFormatter

if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        formatter_class=CustomWidthFormatter,
        description=description,
    )
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
        default=default_folder,
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
        Saved dashboard title, if not provided it will be inferred from
        WIP_DASHBOARD_TITLE with {wip_dashboard_title_suffix!r} suffix removed
        and {ref_dashboard_title_suffix!r} suffix appended as needed.
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
        "wip_dashboard_title",
        metavar="WIP_DASHBOARD_TITLE",
        help=f"""
        The reference or WIP title. The {wip_dashboard_title_suffix!r} suffix will be
        appended as needed.
        """,
    )
    args = parser.parse_args()

    grafana_client = GrafanaClient(
        args.root_url, user=args.user, password=args.password
    )

    out_dir, folder, title = args.out_dir, args.folder, args.title
    if out_dir is None:
        out_dir = os.path.join(
            os.path.dirname(os.path.abspath(__file__)),
            default_out_subdir,
        )

    wip_dashboard_title = args.wip_dashboard_title
    if not wip_dashboard_title.endswith(wip_dashboard_title_suffix):
        wip_dashboard_title += wip_dashboard_title_suffix
    wip_dashboards = grafana_client.search_dashboard(wip_dashboard_title)
    num_dashes = len(wip_dashboards)
    if num_dashes == 0:
        print(f"{wip_dashboard_title!r}: no such dashboard", file=sys.stderr)
        sys.exit(1)
    if num_dashes > 1:
        print(
            f"{wip_dashboard_title!r}: ambiguous, found {num_dashes} matches",
            file=sys.stderr,
        )
        sys.exit(1)
    dashboard_meta = grafana_client.get_dashboard(wip_dashboards[0]["uid"])
    dashboard = dashboard_meta["dashboard"]
    if title is None:
        title = from_to_suffix_title(
            dashboard["title"],
            wip_dashboard_title_suffix,
            ref_dashboard_title_suffix,
        )
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
        f'Dashboard {folder_dash_title(dashboard_meta.get("folderTitle"), wip_dashboard_title)!r} saved into {out_file!r} using {dashboard["title"]!r} title',
        file=sys.stderr,
    )
