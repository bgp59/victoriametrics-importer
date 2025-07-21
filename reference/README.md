<!-- TOC tocDepth:2..3 chapterDepth:2..6 -->

- [Description](#description)
- [Metrics](#metrics)
- [Build And Run Instructions](#build-and-run-instructions)

<!-- /TOC -->

## Description

This an example that can serve as a reference for how to use [victoriametrics-importer](..). The source code and the configuration file are an illustration on how to develop an actual importer.

## Metrics

There are 3 simulated data sources under [parser](parser):

- counter: an unsigned integer incremented by a random amount, min..max, repeated 1..n times, for every scan. This is used for generating 2 metrics: `refvmi_counter_delta`, and `refvmi_counter_rate` (delta/interval).

- gauge: an unsigned integer taking random values, min..max, repeated 1..n times, for every scan. This is used for generating 1 metric: `refvmi_gauge`.

- categorical: a random selection from a given list, min..max, repeated 1..n times, for every scan. This is used for generating 1 metric: `refvmi_categorical`. The category is a value associated with the `choice` label.

All metrics can be configured with a full metrics factor implementing the [Reducing The Number Of Data Points](../README.md#reducing-the-number-of-data-points) approach.

## Build And Run Instructions

```bash
cd reference
./go-build.sh
```

```bash
cd reference
./run-refvmi.sh -use-stdout-metrics-queue       # to display metrics to stdout
```

See local [VMI Infrastructure](../vmi-extras/README.md#vmi-infrastructure), [Docker Support](../vmi-extras/README.md#docker-support) and [Emulated VictoriaMetrics Endpoints](../vmi-extras/README.md#emulated-victoriametrics-endpoints) for examples on how to run it against real or simulated infra.
