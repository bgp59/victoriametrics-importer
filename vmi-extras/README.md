# VMI Extra Tools

<!-- TOC tocDepth:2..3 chapterDepth:2..6 -->

- [Description](#description)
- [Content](#content)
  - [Dev Tools](#dev-tools)
  - [VMI Infrastructure](#vmi-infrastructure)
  - [Docker Support](#docker-support)
  - [Grafana Support](#grafana-support)
  - [Emulated VictoriaMetrics Endpoints](#emulated-victoriametrics-endpoints)

<!-- /TOC -->

## Description

This is a collection of tools that may help with the development of specific VictoriaMetrics importers.

It is lumped with the main VMI module/framework for convenience purpose, but they are independent. Hence a separate versioning, [semver.txt](semver.txt), and tagging, `vmi-extras-vX.Y.Z`, for it.

It can be downloaded from [releases](https://github.com/bgp59/victoriametrics-importer/releases) and normally it should be extracted under the importer's root. If it is extracted elsewhere then `vmi-extras/project-root` symlink should be re-pointed to project's location. This will ensure that the project directory is mounted as a volume by docker containers, allowing it to run code.

## Content

### Dev Tools

[devtools](devtools) is a loose collection of scripts used during VMI development which may come in handy for actual importers. If Python tools will be needed, it is suggested that they are developed under a venv, primed via:

```bash
cd vmi-extras/devtools
./py-prerequisites.sh
 ```

Python files can be formatted via

```bash
./vmi-extras/devtools/py-format.sh [DIR ...]
```

or

```bash
. vmi-extras/devtools/.aliases
pyfmt [DIR ...]
```

### VMI Infrastructure

[vmi-infra](vmi-infra) contains the scripts and files needed to install and run a [Single-node version](https://docs.victoriametrics.com/victoriametrics/single-server-victoriametrics/#) of [VictoriaMetrics](https://docs.victoriametrics.com/) and [Grafana](https://grafana.com/docs/grafana/latest/setup-grafana/?src=ggl-s&mdm=cpc&cnt=99878325494&camp=b-grafana-exac-amer&trm=grafana).

#### Running On A Local Host

**NOTE** Available only for Linux amd64 or arm64.

- install:

    ```bash
    cd vmi-extras/vmi-infra
    ./install-vmi-infra.sh -h
    ```

    ```text
    Usage: install-vmi-infra.sh [-r ROOT_DIR] [-R RUNTIME_DIR]

    Install VictoriaMetrics & Grafana under ROOT_DIR, default: /Users/emy/vmi-infra,
    using RUNTIME_DIR as runtime dir, default: ROOT_DIR/runtime.

    ```

- start / stop:

    ```bash
    cd ~/vmi-infra
    ./start-vmi-infra.sh
    ./stop-vmi-infra.sh
    ```

VictoriaMetrics importer endpoint is `http://HOSTNAME:8428/api/v1/import/prometheus` and Grafana can be accessed at  `http://HOSTNAME:3000`. The Grafana credentials are `admin`/`vmi`

#### Running On A Docker Container

VMI was developed on an MacBook Air M3 running MacOS 15.5.

- build the image

    ```bash
    cd vmi-extras/docker/vmi-infra
    ./build-image
    ```

- start/top the container

    ```bash
    cd vmi-extras/docker/vmi-infra
    ./start-container
    ./stop-container
    ```

    The VictoriaMetrics importer and Grafana ports are exported to the host and therefore the endpoints are the same as above.

- running other code on the container, e.g. VMI's reference:

    ```bash
    cd vmi-extras/docker/vmi-infra
    ./login-container # this will logon on the container
    ```

    ```bash
    # Now on container the project dir is available 
    # under /volumes:
    cd /volumes/victoriametrics-importer/reference
    ./run-refvmi.sh
    ```

### Docker Support

[docker](docker) is a collection of scripts for building images and running containers. Each image is located under its self-explanatory named directory and consists of self-explanatory command scripts and a few supporting files.

#### vmi-base

Generic multi-platform (`linux/amd64` and `linux/arm64`) based on Ubuntu 20.04. A git ignored `volumes` directory will be created at runtime by [pre-start-host-command](vmi-base/pre-start-host-command) hook and it will be mounted as follows on the container:

| Host Path | Container Path | Obs |
|      ---- | ---            | --- |
| volumes/PLATFORM/runtime | /volumes/runtime | PLATFORM: linux/amd64 or linux/arm64 |
| volumes/PROJECT | /volumes/PROJECT | PROJECT -> real path of ../project-root |

#### vmi-infra

multi-platform (`linux/amd64` and `linux/arm64`) based on [vmi-base](#vmi-base), configured to start VictoriaMetrics and Grafana with the respective ports exposed to the host. See [Running On A Docker Container](#running-on-a-docker-container) for details.

It has the same volumes as [vmi-base](#vmi-base). Both VictoriaMetrics and Grafana have their respective state persisted between container restarts because they are stored under `volumes/PLATFORM/runtime` on the host. To make a fresh start remove the relevant runtime sub-directory **before** starting the container.

### Grafana Support

Grafana setup contains [provisioned dashboards](https://grafana.com/docs/grafana/latest/administration/provisioning/#dashboards), stored under [vmi-infra/files/update/grafana/dashboards](vmi-infra/files/update/grafana/dashboards). Included there is `internal-metrics-ref`, a reference dashboard for VMI [Internal Metrics](../docs/internal_metrics.md).

When developing actual importers, it is a good practice to create provisioned reference dashboards illustrating the available metrics. Such dashboards should be included in Demo/PoC Docker images, the latter based on [vmi-infra](#vmi-infra).

However such dashboards cannot be edited in place, first a copy should be made and then, once the changes are completed, it should be downloaded to the proper location and its title should be updated.

The following Python tools can help w/ the above:

```bash
cd vmi-extras/grafana
./prepare-grafana-wip-dashboard.py -h
```

```text
usage: prepare-grafana-wip-dashboard.py [-h] [-r ROOT_URL] [-u USER]
                                        [-p PASSWORD]
                                        DASHBOARD_TITLE

positional arguments:
  DASHBOARD_TITLE

optional arguments:
  -h, --help            show this help message and exit
  -r ROOT_URL, --root-url ROOT_URL
                        Grafana root URL, default: 'http://localhost:3000'.
  -u USER, --user USER  Grafana user, default: 'admin'.
  -p PASSWORD, --password PASSWORD
                        Grafana password, default: 'vmi'.
```

```bash
cd vmi-extras/grafana
./save-grafana-wip-dashboard.py -h
```

```text
usage: save-grafana-wip-dashboard.py [-h] [-r ROOT_URL] [-u USER]
                                     [-p PASSWORD] [-f FOLDER] [-k KEEP]
                                     [-t TITLE] [-o OUT_DIR]
                                     DASHBOARD_TITLE

positional arguments:
  DASHBOARD_TITLE       The reference or WIP title. The ' (WIP)' suffix will
                        be appended as needed.

optional arguments:
  -h, --help            show this help message and exit
  -r ROOT_URL, --root-url ROOT_URL
                        Grafana root URL, default: 'http://localhost:3000'.
  -u USER, --user USER  Grafana user, default: 'admin'.
  -p PASSWORD, --password PASSWORD
                        Grafana password, default: 'vmi'.
  -f FOLDER, --folder FOLDER
                        Grafana folder, default: 'vmi-reference'.
  -k KEEP, --keep KEEP  Keep Instance and Hostname var selection. By default
                        they are either cleared or set to All if the latter is
                        enabled.
  -t TITLE, --title TITLE
                        New title, if not provided it will be inferred from
                        WIP_DASHBOARD_TITLE with ' (WIP)' suffix removed and
                        '-ref' suffix appended as needed.
  -o OUT_DIR, --out-dir OUT_DIR
                        Output dir, default is the location of this
                        script/dashboards
```

### Emulated VictoriaMetrics Endpoints

[vmi-endpoint](vmi-endpoint) offers a HTTP server that accepts HTTP PUT requests and optionally displays them  to stdout or saves them to a file.

It can be used to simulate failures and to exercise failover mechanism.

- build:

    ```bash
    cd vmi-extras/vmi-endpoint
    ./go-build.sh
    ```

- args:

    ```text
    Usage of ./vmi-endpoint:
    -audit-file string
            Audit file, use `-' for stdout
    -bind-addr string
            Listen bind address (default "localhost")
    -display-body-limit int
            Display only the first N bytes of the body, use 0 for no limit (default 512)
    -display-level string
            Display level, one of: ["request" "headers" "body"]
    -port string
            Listen port (default "8080")
    -traffic-stats-int string
            Traffic stats interval, use 0 to disable (default "0")
    ```

- invocation wrapper:

    ```bash
    cd vmi-extras/vmi-endpoint
    ./run-vmi-endpoint.sh -h
    ```

    ```text
    Usage: run-vmi-endpoint.sh RUN PAUSE [ARG...]
    Run vmi-endpoint ARG... in a loop, RUN sec active, PAUSE down

    For vmi-endpoint invoke run-vmi-endpoint.sh -H | --vmi-endpoint-help
    ```

For instance:

- start 2 endpoints with different RUN/PAUSE cycles:
  - #1

    ```bash
    cd vmi-extras/vmi-endpoint
    ./run-vmi-endpoint.sh 13 8 \
        -port 8081 \
        -display-level body \
        -display-body-limit 0
    ```

  - #2

    ```bash
    cd vmi-extras/vmi-endpoint
    ./run-vmi-endpoint.sh 8 13 \
        -port 8082 \
        -display-level body \
        -display-body-limit 0
    ```

- start the reference importer, pointed to both of them:

    ```bash
    cd reference
    ./run-refvmi.sh -http-pool-endpoints \
        http://localhost:8081,http://localhost:8082
    ```
