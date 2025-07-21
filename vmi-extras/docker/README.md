# Docker Utilities

## Background Info

This directory contains the tools for creating and running [Docker](https://www.docker.com/) containers for development under non Linux platforms and for demo PoC.

Each image is supported by its own sub-directory:

* `base`: for Linux base
* `dev`: for [developing under a different OS](../../../docs/dev_guide.md#developing-under-a-different-os)
* `demo`: for [running the PoC using a containerized solution](../../../docs/poc.md#using-a-containerized-solution)

## Multi-platform v. Single Platform Images

The Docker [multi-platform](https://docs.docker.com/build/building/multi-platform/) images are preferred. However single platform images are also supported and the suffix -_OS_-_ARCH_ will be added to their base name.

### How To Build, Use And Publish Multi-platform Images

* use:

    ```bash
    cd dev
    ./build-image multi
    ./start-multi-container

    ./run-lsvmi-in-container.sh

    ./stop-container
    ```

* publish:

    ```bash
    cd demo
    ./build-image multi
    ./push-latest-image multi
    ```

### How To Build, Use And Publish Single Platform Images

* use:

    ```bash
    cd dev
    ./build-image       # os/arch
    ./start-container   # --platform os/arch

    ./run-lsvmi-in-container.sh

    ./stop-container
    ```

* publish:

    ```bash
    cd demo
    ./build-image all   # os/arch
    ./push-latest-image all    # os/arch
    ```
