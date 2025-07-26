# VMI Internal Metrics (id: `internal_metrics`)

<!-- TOC tocDepth:2..6 chapterDepth:2..6 -->

- [General Information](#general-information)
- [Agent Metrics](#agent-metrics)
  - [vmi_uptime_sec](#vmi_uptime_sec)
  - [vmi_buildinfo](#vmi_buildinfo)
  - [vmi_proc_pcpu](#vmi_proc_pcpu)
- [Compressor Pool Metrics](#compressor-pool-metrics)
  - [vmi_compressor_read_delta](#vmi_compressor_read_delta)
  - [vmi_compressor_read_byte_delta](#vmi_compressor_read_byte_delta)
  - [vmi_compressor_send_delta](#vmi_compressor_send_delta)
  - [vmi_compressor_send_byte_delta](#vmi_compressor_send_byte_delta)
  - [vmi_compressor_send_error_delta](#vmi_compressor_send_error_delta)
  - [vmi_compressor_tout_flush_delta](#vmi_compressor_tout_flush_delta)
  - [vmi_compressor_write_error_delta](#vmi_compressor_write_error_delta)
  - [vmi_compressor_compression_factor](#vmi_compressor_compression_factor)
- [Generator Metrics](#generator-metrics)
  - [vmi_metrics_gen_invocation_delta](#vmi_metrics_gen_invocation_delta)
  - [vmi_metrics_gen_metrics_delta](#vmi_metrics_gen_metrics_delta)
  - [vmi_metrics_gen_byte_delta](#vmi_metrics_gen_byte_delta)
  - [vmi_metrics_gen_dtime_sec](#vmi_metrics_gen_dtime_sec)
- [Go Specific Metrics](#go-specific-metrics)
  - [vmi_go_mem_free_delta](#vmi_go_mem_free_delta)
  - [vmi_go_mem_gc_delta](#vmi_go_mem_gc_delta)
  - [vmi_go_mem_malloc_delta](#vmi_go_mem_malloc_delta)
  - [vmi_go_num_goroutine](#vmi_go_num_goroutine)
  - [vmi_go_mem_in_use_object_count](#vmi_go_mem_in_use_object_count)
  - [vmi_go_mem_heap_bytes](#vmi_go_mem_heap_bytes)
  - [vmi_go_mem_heap_sys_bytes](#vmi_go_mem_heap_sys_bytes)
  - [vmi_go_mem_sys_bytes](#vmi_go_mem_sys_bytes)
- [HTTP Endpoint Pool Metrics](#http-endpoint-pool-metrics)
  - [Per Endpoint Metrics](#per-endpoint-metrics)
    - [vmi_http_ep_send_buffer_delta](#vmi_http_ep_send_buffer_delta)
    - [vmi_http_ep_send_buffer_byte_delta](#vmi_http_ep_send_buffer_byte_delta)
    - [vmi_http_ep_send_buffer_error_delta](#vmi_http_ep_send_buffer_error_delta)
    - [vmi_http_ep_healthcheck_delta](#vmi_http_ep_healthcheck_delta)
    - [vmi_http_ep_healthcheck_error_delta](#vmi_http_ep_healthcheck_error_delta)
  - [Per Pool Metrics](#per-pool-metrics)
    - [vmi_http_ep_pool_healthy_rotate_count](#vmi_http_ep_pool_healthy_rotate_count)
    - [vmi_http_ep_pool_no_healthy_ep_error_delta](#vmi_http_ep_pool_no_healthy_ep_error_delta)
- [OS Metrics](#os-metrics)
  - [vmi_os_info](#vmi_os_info)
  - [vmi_os_release](#vmi_os_release)
  - [vmi_os_uptime_sec](#vmi_os_uptime_sec)
- [Scheduler Metrics](#scheduler-metrics)
  - [vmi_task_scheduled_delta](#vmi_task_scheduled_delta)
  - [vmi_task_delayed_delta](#vmi_task_delayed_delta)
  - [vmi_task_overrun_delta](#vmi_task_overrun_delta)
  - [vmi_task_executed_delta](#vmi_task_executed_delta)
  - [vmi_task_deadline_hack_delta](#vmi_task_deadline_hack_delta)
  - [vmi_task_interval_avg_runtime_sec](#vmi_task_interval_avg_runtime_sec)

<!-- /TOC -->

## General Information

These are metrics relating to the agent itself. There is no partial/full cycle approach for these metrics, the entire set is generated for every cycle.

## Agent Metrics

**NOTE!** Unless otherwise stated, the metrics in this paragraph have the following label set:

  | Label Name | Value(s)/Info |
  | --- | --- |
  | vmi_inst | _instance_ |
  | hostname | _hostname_ |

### vmi_uptime_sec

Time, in seconds, since the agent was started.
  
  | Label Name | Value(s)/Info |
  | --- | --- |
  | vmi_inst | _instance_ |
  | hostname | _hostname_ |

### vmi_buildinfo

Categorical metric (constant `1`) with build info:

**NOTE!** Generated for full cycle only.

  | Label Name | Value(s)/Info |
  | --- | --- |
  | vmi_inst | _instance_ |
  | hostname | _hostname_ |
  | version | semver of the agent |
  | gitinfo | _commit-id_\[-dirty\] |

### vmi_proc_pcpu

The %CPU for the scan interval.

## Compressor Pool Metrics

**NOTE!** Unless otherwise stated, the metrics in this paragraph have the following label set:

  | Label Name | Value(s)/Info |
  | --- | --- |
  | vmi_inst | _instance_ |
  | hostname | _hostname_ |
  | compressor | _compressor#_ (0 .. num_compressor - 1) |

### vmi_compressor_read_delta

The number of reads from the queue since the last scan.

### vmi_compressor_read_byte_delta

The number of read bytes from the queue since the last scan.

### vmi_compressor_send_delta

The number of sends since the last scan.

### vmi_compressor_send_byte_delta

The number of sent bytes since the last scan.

### vmi_compressor_send_error_delta

The number of send errors since the last scan.

### vmi_compressor_tout_flush_delta

The number of timeout (timed based, that is) flushes since the last scan.

### vmi_compressor_write_error_delta

The number of write (to compressor stream) errors since the last scan.

### vmi_compressor_compression_factor

The (exponentially decaying) compression factor average.

## Generator Metrics

Each metrics generator maintains a standard set of stats, updated at the start/end of the generator's invocation.

**NOTE!** Unless otherwise stated, the metrics in this paragraph have the following label set:

  | Label Name | Value(s)/Info |
  | --- | --- |
  | vmi_inst | _instance_ |
  | hostname | _hostname_ |
  | gen_id | _id_ the unique generator ID e.g. `internal_metrics`, `proc_pid_metrics` |

### vmi_metrics_gen_invocation_delta

The invocation count delta, computed for the internal metrics scan interval.

### vmi_metrics_gen_metrics_delta

The metric count delta, computed for the internal metrics scan interval. This can be less than the theoretical count due to [Reducing The Number Of Data Points](../README.md#reducing-the-number-of-data-points) techniques.

### vmi_metrics_gen_byte_delta

The byte delta, computed for the internal metrics scan interval.

### vmi_metrics_gen_dtime_sec

The actual time delta, in seconds, since the previous invocation. Theoretically this should be close to the configured interval interval, but it may vary, especially on loaded systems. This can be used for computing rates out of deltas.

## Go Specific Metrics

**NOTE!** Unless otherwise stated, the metrics in this paragraph have the following label set:

  | Label Name | Value(s)/Info |
  | --- | --- |
  | vmi_inst | _instance_ |
  | hostname | _hostname_ |

### vmi_go_mem_free_delta

The number of `free` calls since the last scan.

### vmi_go_mem_gc_delta

The number of garbage collector calls since the last scan.

### vmi_go_mem_malloc_delta

The number of `malloc` calls since the last scan.

### vmi_go_num_goroutine

The current number of goroutines.

### vmi_go_mem_in_use_object_count

The current number of go objects in use.

### vmi_go_mem_heap_bytes

### vmi_go_mem_heap_sys_bytes

### vmi_go_mem_sys_bytes

The size of various memory pools, in bytes.

## HTTP Endpoint Pool Metrics

### Per Endpoint Metrics

**NOTE!** Unless otherwise stated, the metrics in this paragraph have the following label set:

  | Label Name | Value(s)/Info |
  | --- | --- |
  | vmi_inst | _instance_ |
  | hostname | _hostname_ |
  | url | _url_ |

#### vmi_http_ep_send_buffer_delta

The number of send calls against this URL, since the last scan.

#### vmi_http_ep_send_buffer_byte_delta

The number of bytes sent to this URL, since the last scan.

#### vmi_http_ep_send_buffer_error_delta

The number of send call errors against this URL, since the last scan.

#### vmi_http_ep_healthcheck_delta

The number of health checks for this URL, since the last scan.

#### vmi_http_ep_healthcheck_error_delta

The number of failed health checks for this URL, since the last scan.

### Per Pool Metrics

**NOTE!** Unless otherwise stated, the metrics in this paragraph have the following label set:

  | Label Name | Value(s)/Info |
  | --- | --- |
  | vmi_inst | _instance_ |
  | hostname | _hostname_ |

#### vmi_http_ep_pool_healthy_rotate_count

The cumulative number of rotations.

#### vmi_http_ep_pool_no_healthy_ep_error_delta

The number of endpoint errors since the last scan.

## OS Metrics

**NOTE!** Unless otherwise stated, the metrics in this paragraph have the following label set:

  | Label Name | Value(s)/Info |
  | --- | --- |
  | vmi_inst | _instance_ |
  | hostname | _hostname_ |

### vmi_os_info

Categorical metric (constant `1`) with Unix [uname](https://linux.die.net/man/1/uname) info:

**NOTE!** Generated for full cycle only.

  | Label Name | Value(s)/Info |
  | --- | --- |
  | vmi_inst | _instance_ |
  | hostname | _hostname_ |
  | name | \`uname -s\` |
  | release | \`uname -r\` |
  | version | \`uname -v\` |
  | machine | \`uname -m\` |

### vmi_os_release

Categorical metric (constant `1`) with [os-release](https://man7.org/linux/man-pages/man5/os-release.5.html) info:

**NOTE!** Generated for full cycle only.

  | Label Name | Value(s)/Info |
  | --- | --- |
  | vmi_inst | _instance_ |
  | hostname | _hostname_ |
  | id<br>name<br>pretty_name<br>version<br>version_codename<br>version_id | See the eponymous fields in lower case from [os-release](https://man7.org/linux/man-pages/man5/os-release.5.html) |

  **Note!** This is Linux specific for now, is will not be populated for other OS'es.

### vmi_os_uptime_sec

  Time, in seconds, since OS boot

## Scheduler Metrics

**NOTE!** They all have the same label set:

  | Label Name | Value(s)/Info |
  | --- | --- |
  | vmi_inst | _instance_ |
  | hostname | _hostname_ |
  | task_id | _id_, e.g. `internal_metrics`, `proc_pid_metrics`, etc |

### vmi_task_scheduled_delta

The number of times the task was scheduled, since previous scan.

### vmi_task_delayed_delta

The number of times the task was delayed because its next reschedule would have been too close to the deadline, since the last scan.

### vmi_task_overrun_delta

The number of times the task ran past the next  deadline, since the last scan.

### vmi_task_executed_delta

The number of times the task was executed, since the last scan.

### vmi_task_deadline_hack_delta

The number of times the deadline hack was applied for the task, since the last scan.

The hack is required for a rare condition observed when running Docker on a MacBook whereby the clock appears to move backwards and the next deadline results in being before the previous one. The hack consist in adding task intervals until the chronological order is restored.

### vmi_task_interval_avg_runtime_sec

The average time, in seconds, for all the runs of the task, since the last scan.
