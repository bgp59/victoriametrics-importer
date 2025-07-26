// All internal metrics definitions in one place

package vmi_internal

const (
	// The following labels are common to all metrics:
	INSTANCE_LABEL_NAME = "vmi_inst"
	HOSTNAME_LABEL_NAME = "hostname"

	//////////////////////////////////////////////////////
	//  Compressor Pool Metrics
	//////////////////////////////////////////////////////

	// Deltas since previous internal metrics interval:
	COMPRESSOR_STATS_READ_DELTA_METRIC          = "vmi_compressor_read_delta"
	COMPRESSOR_STATS_READ_BYTE_DELTA_METRIC     = "vmi_compressor_read_byte_delta"
	COMPRESSOR_STATS_SEND_DELTA_METRIC          = "vmi_compressor_send_delta"
	COMPRESSOR_STATS_SEND_BYTE_DELTA_METRIC     = "vmi_compressor_send_byte_delta"
	COMPRESSOR_STATS_TIMEOUT_FLUSH_DELTA_METRIC = "vmi_compressor_tout_flush_delta"
	COMPRESSOR_STATS_SEND_ERROR_DELTA_METRIC    = "vmi_compressor_send_error_delta"
	COMPRESSOR_STATS_WRITE_ERROR_DELTA_METRIC   = "vmi_compressor_write_error_delta"
	COMPRESSOR_STATS_COMPRESSION_FACTOR_METRIC  = "vmi_compressor_compression_factor"

	COMPRESSOR_ID_LABEL_NAME = "compressor"

	//////////////////////////////////////////////////////
	// Generator Metrics
	//////////////////////////////////////////////////////

	// Invocation, metric and byte counts for the generator:
	METRICS_GENERATOR_INVOCATION_DELTA_METRIC = "vmi_metrics_gen_invocation_delta"
	METRICS_GENERATOR_METRICS_DELTA_METRIC    = "vmi_metrics_gen_metrics_delta"
	METRICS_GENERATOR_BYTE_DELTA_METRIC       = "vmi_metrics_gen_byte_delta"

	// Actual interval since the previous invocation. It should be closed to the
	// configured interval, but may be longer if the generator is busy. It could
	// be used to calculate the rates out of deltas
	METRICS_GENERATOR_DTIME_METRIC           = "vmi_metrics_gen_dtime_sec"
	METRICS_GENERATOR_DTIME_METRIC_PRECISION = 6

	METRICS_GENERATOR_ID_LABEL_NAME = "gen_id"

	//////////////////////////////////////////////////////
	// Go Metrics
	//////////////////////////////////////////////////////

	GO_NUM_GOROUTINE_METRIC           = "vmi_go_num_goroutine"
	GO_MEM_SYS_BYTES_METRIC           = "vmi_go_mem_sys_bytes"
	GO_MEM_HEAP_BYTES_METRIC          = "vmi_go_mem_heap_bytes"
	GO_MEM_HEAP_SYS_BYTES_METRIC      = "vmi_go_mem_heap_sys_bytes"
	GO_MEM_IN_USE_OBJECT_COUNT_METRIC = "vmi_go_mem_in_use_object_count"

	// Deltas since previous internal metrics interval:
	GO_MEM_MALLOCS_DELTA_METRIC = "vmi_go_mem_malloc_delta"
	GO_MEM_FREE_DELTA_METRIC    = "vmi_go_mem_free_delta"
	GO_MEM_NUM_GC_DELTA_METRIC  = "vmi_go_mem_gc_delta"

	//////////////////////////////////////////////////////
	// HTTP Endpoint Pool Metrics
	//////////////////////////////////////////////////////

	// Per endpoint:

	// Deltas since previous internal metrics interval:
	HTTP_ENDPOINT_STATS_SEND_BUFFER_DELTA_METRIC        = "vmi_http_ep_send_buffer_delta"
	HTTP_ENDPOINT_STATS_SEND_BUFFER_BYTE_DELTA_METRIC   = "vmi_http_ep_send_buffer_byte_delta"
	HTTP_ENDPOINT_STATS_SEND_BUFFER_ERROR_DELTA_METRIC  = "vmi_http_ep_send_buffer_error_delta"
	HTTP_ENDPOINT_STATS_HEALTH_CHECK_DELTA_METRIC       = "vmi_http_ep_healthcheck_delta"
	HTTP_ENDPOINT_STATS_HEALTH_CHECK_ERROR_DELTA_METRIC = "vmi_http_ep_healthcheck_error_delta"

	// Labels:
	HTTP_ENDPOINT_STATS_STATE_LABEL = "state"
	HTTP_ENDPOINT_URL_LABEL_NAME    = "url"

	// Per pool:

	// Deltas since previous internal metrics interval:
	HTTP_ENDPOINT_POOL_STATS_HEALTHY_ROTATE_DELTA_METRIC      = "vmi_http_ep_pool_healthy_rotate_delta"
	HTTP_ENDPOINT_POOL_STATS_NO_HEALTHY_EP_ERROR_DELTA_METRIC = "vmi_http_ep_pool_no_healthy_ep_error_delta"

	//////////////////////////////////////////////////////
	// Importer Metrics
	//////////////////////////////////////////////////////

	// Importer metric:
	VMI_UPTIME_METRIC = "vmi_uptime_sec" // heartbeat

	VMI_BUILDINFO_METRIC    = "vmi_buildinfo"
	VMI_VERSION_LABEL_NAME  = "vmi_version"
	VMI_GIT_INFO_LABEL_NAME = "vmi_git_info"

	// OS metrics:
	OS_INFO_METRIC          = "vmi_os_info"
	OS_INFO_LABEL_PREFIX    = "os_info_" // prefix + OSInfoLabelKeys
	OS_RELEASE_METRIC       = "vmi_os_release"
	OS_RELEASE_LABEL_PREFIX = "os_rel_" // prefix + OSReleaseLabelKeys
	OS_UPTIME_METRIC        = "vmi_os_uptime_sec"

	UPTIME_METRIC_PRECISION = 6

	//////////////////////////////////////////////////////
	// Process Metrics
	//////////////////////////////////////////////////////

	// %CPU over internal metrics interval:
	VMI_PROC_PCPU_METRIC = "vmi_proc_pcpu"

	//////////////////////////////////////////////////////
	// Task Scheduler Metrics
	//////////////////////////////////////////////////////

	TASK_STATS_SCHEDULED_DELTA_METRIC       = "vmi_task_scheduled_delta"
	TASK_STATS_DELAYED_DELTA_METRIC         = "vmi_task_delayed_delta"
	TASK_STATS_OVERRUN_DELTA_METRIC         = "vmi_task_overrun_delta"
	TASK_STATS_EXECUTED_DELTA_METRIC        = "vmi_task_executed_delta"
	TASK_STATS_NEXT_TS_HACK_DELTA_METRIC    = "vmi_task_next_ts_hack_delta"
	TASK_STATS_AVG_RUNTIME_METRIC           = "vmi_task_avg_runtime_sec"
	TASK_STATS_AVG_RUNTIME_METRIC_PRECISION = 6

	// Re-use generator ID label since they have the same value:
	TASK_STATS_TASK_ID_LABEL_NAME = METRICS_GENERATOR_ID_LABEL_NAME
)
