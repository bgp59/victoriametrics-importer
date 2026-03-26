#! /usr/bin/env python3

description = """
All VMI reference dashboards are provisioned so they cannot be changed in place.
This script will make an editable copy under the work-in-progress folder.
"""

import argparse
import sys

from definitions import (
    default_grafana_password,
    default_grafana_root_url,
    default_grafana_user,
    ref_dashboard_title_suffix,
    wip_dashboard_title_suffix,
    wip_folder_title,
)
from grafana import (
    GrafanaClient,
    folder_dash_title,
    from_to_suffix_title,
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
        "dashboard_title",
        metavar="DASHBOARD_TITLE",
    )
    args = parser.parse_args()
    grafana_client = GrafanaClient(
        args.root_url, user=args.user, password=args.password
    )

    ref_dashboard_title = args.dashboard_title

    ref_dashboards = grafana_client.search_dashboard(ref_dashboard_title)
    num_dashes = len(ref_dashboards)
    if num_dashes == 0:
        print(f"{ref_dashboard_title!r}: no such dashboard", file=sys.stderr)
        sys.exit(1)
    if num_dashes > 1:
        print(
            f"{ref_dashboard_title!r}: ambiguous, found {num_dashes} matches",
            file=sys.stderr,
        )
        sys.exit(1)
    ref_uid = ref_dashboards[0]["uid"]

    ref_dashboard = grafana_client.get_dashboard(ref_uid)
    dashboard, meta = ref_dashboard["dashboard"], ref_dashboard["meta"]
    dashboard["title"] = from_to_suffix_title(
        dashboard["title"],
        from_suffix=ref_dashboard_title_suffix,
        to_suffix=wip_dashboard_title_suffix,
    )
    dashboard["id"] = None
    dashboard["uid"] = None

    wip_folders = grafana_client.search_folder(wip_folder_title)
    num_folders = len(wip_folders)
    if num_folders == 0:
        wip_folder = grafana_client.create_folder(wip_folder_title)
        wip_folder_uid = wip_folder["uid"]
    elif num_folders == 1:
        wip_folder_uid = wip_folders[0]["uid"]
    else:
        print(
            f"{wip_folder_title!r}: multiple ({num_folders}) found, remove all but one",
            file=sys.stderr,
        )
        sys.exit(1)

    from_folder_dash_title = folder_dash_title(
        meta.get("folderTitle"), ref_dashboard_title
    )

    grafana_client.save_dashboard(
        dashboard,
        folder_uid=wip_folder_uid,
        message=f"from {from_folder_dash_title}",
    )
    print(
        f'Created {folder_dash_title(wip_folder_title, dashboard.get("title"))!r} from {from_folder_dash_title!r}',
        file=sys.stderr,
    )
