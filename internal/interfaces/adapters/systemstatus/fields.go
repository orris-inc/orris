// Package systemstatus provides shared utilities for system status Redis operations.
package systemstatus

// Redis hash field names for system status
const (
	FieldCPUPercent    = "cpu_percent"
	FieldMemoryPercent = "memory_percent"
	FieldMemoryUsed    = "memory_used"
	FieldMemoryTotal   = "memory_total"
	FieldMemoryAvail   = "memory_avail"
	FieldDiskPercent   = "disk_percent"
	FieldDiskUsed      = "disk_used"
	FieldDiskTotal     = "disk_total"
	FieldUptimeSeconds = "uptime_seconds"

	FieldLoadAvg1  = "load_avg_1"
	FieldLoadAvg5  = "load_avg_5"
	FieldLoadAvg15 = "load_avg_15"

	FieldNetworkRxBytes = "network_rx_bytes"
	FieldNetworkTxBytes = "network_tx_bytes"
	FieldNetworkRxRate  = "network_rx_rate"
	FieldNetworkTxRate  = "network_tx_rate"

	FieldTCPConnections = "tcp_connections"
	FieldUDPConnections = "udp_connections"

	FieldPublicIPv4 = "public_ipv4"
	FieldPublicIPv6 = "public_ipv6"

	FieldAgentVersion = "agent_version"
	FieldPlatform     = "platform"
	FieldArch         = "arch"

	FieldUpdatedAt = "updated_at"

	// CPU details
	FieldCPUCores     = "cpu_cores"
	FieldCPUModelName = "cpu_model_name"
	FieldCPUMHz       = "cpu_mhz"

	// Swap memory
	FieldSwapTotal   = "swap_total"
	FieldSwapUsed    = "swap_used"
	FieldSwapPercent = "swap_percent"

	// Disk I/O
	FieldDiskReadBytes  = "disk_read_bytes"
	FieldDiskWriteBytes = "disk_write_bytes"
	FieldDiskReadRate   = "disk_read_rate"
	FieldDiskWriteRate  = "disk_write_rate"
	FieldDiskIOPS       = "disk_iops"

	// Pressure Stall Information (PSI)
	FieldPSICPUSome    = "psi_cpu_some"
	FieldPSICPUFull    = "psi_cpu_full"
	FieldPSIMemorySome = "psi_memory_some"
	FieldPSIMemoryFull = "psi_memory_full"
	FieldPSIIOSome     = "psi_io_some"
	FieldPSIIOFull     = "psi_io_full"

	// Network extended stats
	FieldNetworkRxPackets = "network_rx_packets"
	FieldNetworkTxPackets = "network_tx_packets"
	FieldNetworkRxErrors  = "network_rx_errors"
	FieldNetworkTxErrors  = "network_tx_errors"
	FieldNetworkRxDropped = "network_rx_dropped"
	FieldNetworkTxDropped = "network_tx_dropped"

	// Socket statistics
	FieldSocketsUsed      = "sockets_used"
	FieldSocketsTCPInUse  = "sockets_tcp_in_use"
	FieldSocketsUDPInUse  = "sockets_udp_in_use"
	FieldSocketsTCPOrphan = "sockets_tcp_orphan"
	FieldSocketsTCPTW     = "sockets_tcp_tw"

	// Process statistics
	FieldProcessesTotal   = "processes_total"
	FieldProcessesRunning = "processes_running"
	FieldProcessesBlocked = "processes_blocked"

	// File descriptors
	FieldFileNrAllocated = "file_nr_allocated"
	FieldFileNrMax       = "file_nr_max"

	// Context switches and interrupts
	FieldContextSwitches = "context_switches"
	FieldInterrupts      = "interrupts"

	// Kernel info
	FieldKernelVersion = "kernel_version"
	FieldHostname      = "hostname"

	// Virtual memory statistics
	FieldVMPageIn  = "vm_page_in"
	FieldVMPageOut = "vm_page_out"
	FieldVMSwapIn  = "vm_swap_in"
	FieldVMSwapOut = "vm_swap_out"
	FieldVMOOMKill = "vm_oom_kill"

	// Entropy pool
	FieldEntropyAvailable = "entropy_available"
)
