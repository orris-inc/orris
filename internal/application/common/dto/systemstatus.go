// Package dto provides common data transfer objects shared across domains.
package dto

// SystemStatus contains common system metrics shared by all agent types
// (Forward Agent and Node Agent).
type SystemStatus struct {
	// System resources
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryPercent float64 `json:"memory_percent"`
	MemoryUsed    uint64  `json:"memory_used"`
	MemoryTotal   uint64  `json:"memory_total"`
	MemoryAvail   uint64  `json:"memory_avail"`
	DiskPercent   float64 `json:"disk_percent"`
	DiskUsed      uint64  `json:"disk_used"`
	DiskTotal     uint64  `json:"disk_total"`
	UptimeSeconds int64   `json:"uptime_seconds"`

	// System load
	LoadAvg1  float64 `json:"load_avg_1"`
	LoadAvg5  float64 `json:"load_avg_5"`
	LoadAvg15 float64 `json:"load_avg_15"`

	// Network statistics
	NetworkRxBytes uint64 `json:"network_rx_bytes"`
	NetworkTxBytes uint64 `json:"network_tx_bytes"`
	NetworkRxRate  uint64 `json:"network_rx_rate"`
	NetworkTxRate  uint64 `json:"network_tx_rate"`

	// Connection statistics
	TCPConnections int `json:"tcp_connections"`
	UDPConnections int `json:"udp_connections"`

	// Network info
	PublicIPv4 string `json:"public_ipv4,omitempty"`
	PublicIPv6 string `json:"public_ipv6,omitempty"`

	// Agent info
	AgentVersion string `json:"agent_version,omitempty"`
	Platform     string `json:"platform,omitempty"`
	Arch         string `json:"arch,omitempty"`

	// CPU details
	CPUCores     int     `json:"cpu_cores"`
	CPUModelName string  `json:"cpu_model_name"`
	CPUMHz       float64 `json:"cpu_mhz"`

	// Swap memory
	SwapTotal   uint64  `json:"swap_total"`
	SwapUsed    uint64  `json:"swap_used"`
	SwapPercent float64 `json:"swap_percent"`

	// Disk I/O
	DiskReadBytes  uint64 `json:"disk_read_bytes"`
	DiskWriteBytes uint64 `json:"disk_write_bytes"`
	DiskReadRate   uint64 `json:"disk_read_rate"`
	DiskWriteRate  uint64 `json:"disk_write_rate"`
	DiskIOPS       uint64 `json:"disk_iops"`

	// Pressure Stall Information (PSI)
	PSICPUSome    float64 `json:"psi_cpu_some"`
	PSICPUFull    float64 `json:"psi_cpu_full"`
	PSIMemorySome float64 `json:"psi_memory_some"`
	PSIMemoryFull float64 `json:"psi_memory_full"`
	PSIIOSome     float64 `json:"psi_io_some"`
	PSIIOFull     float64 `json:"psi_io_full"`

	// Network extended stats
	NetworkRxPackets uint64 `json:"network_rx_packets"`
	NetworkTxPackets uint64 `json:"network_tx_packets"`
	NetworkRxErrors  uint64 `json:"network_rx_errors"`
	NetworkTxErrors  uint64 `json:"network_tx_errors"`
	NetworkRxDropped uint64 `json:"network_rx_dropped"`
	NetworkTxDropped uint64 `json:"network_tx_dropped"`

	// Socket statistics
	SocketsUsed      int `json:"sockets_used"`
	SocketsTCPInUse  int `json:"sockets_tcp_in_use"`
	SocketsUDPInUse  int `json:"sockets_udp_in_use"`
	SocketsTCPOrphan int `json:"sockets_tcp_orphan"`
	SocketsTCPTW     int `json:"sockets_tcp_tw"`

	// Process statistics
	ProcessesTotal   uint64 `json:"processes_total"`
	ProcessesRunning uint64 `json:"processes_running"`
	ProcessesBlocked uint64 `json:"processes_blocked"`

	// File descriptors
	FileNrAllocated uint64 `json:"file_nr_allocated"`
	FileNrMax       uint64 `json:"file_nr_max"`

	// Context switches and interrupts
	ContextSwitches uint64 `json:"context_switches"`
	Interrupts      uint64 `json:"interrupts"`

	// Kernel info
	KernelVersion string `json:"kernel_version"`
	Hostname      string `json:"hostname"`

	// Virtual memory statistics
	VMPageIn  uint64 `json:"vm_page_in"`
	VMPageOut uint64 `json:"vm_page_out"`
	VMSwapIn  uint64 `json:"vm_swap_in"`
	VMSwapOut uint64 `json:"vm_swap_out"`
	VMOOMKill uint64 `json:"vm_oom_kill"`

	// Entropy pool
	EntropyAvailable uint64 `json:"entropy_available"`

	// Metadata
	UpdatedAt int64 `json:"updated_at,omitempty"`
}
