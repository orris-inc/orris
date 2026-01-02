package systemstatus

import (
	"fmt"

	commondto "github.com/orris-inc/orris/internal/application/common/dto"
)

// String field length limits to prevent memory exhaustion attacks
const (
	MaxIPLength           = 45  // IPv6 max length
	MaxVersionLength      = 32  // Agent version, platform, arch
	MaxCPUModelNameLength = 128 // CPU model name
	MaxKernelVersionLen   = 128 // Kernel version
	MaxHostnameLength     = 255 // RFC 1035 hostname limit
)

// truncateString truncates a string to the specified max length.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

// ParseSystemStatus parses common system status fields from Redis hash values.
func ParseSystemStatus(values map[string]string) commondto.SystemStatus {
	var status commondto.SystemStatus

	// Float fields
	fmt.Sscanf(values[FieldCPUPercent], "%f", &status.CPUPercent)
	fmt.Sscanf(values[FieldMemoryPercent], "%f", &status.MemoryPercent)
	fmt.Sscanf(values[FieldDiskPercent], "%f", &status.DiskPercent)
	fmt.Sscanf(values[FieldLoadAvg1], "%f", &status.LoadAvg1)
	fmt.Sscanf(values[FieldLoadAvg5], "%f", &status.LoadAvg5)
	fmt.Sscanf(values[FieldLoadAvg15], "%f", &status.LoadAvg15)

	// Uint64 fields
	fmt.Sscanf(values[FieldMemoryUsed], "%d", &status.MemoryUsed)
	fmt.Sscanf(values[FieldMemoryTotal], "%d", &status.MemoryTotal)
	fmt.Sscanf(values[FieldMemoryAvail], "%d", &status.MemoryAvail)
	fmt.Sscanf(values[FieldDiskUsed], "%d", &status.DiskUsed)
	fmt.Sscanf(values[FieldDiskTotal], "%d", &status.DiskTotal)
	fmt.Sscanf(values[FieldNetworkRxBytes], "%d", &status.NetworkRxBytes)
	fmt.Sscanf(values[FieldNetworkTxBytes], "%d", &status.NetworkTxBytes)
	fmt.Sscanf(values[FieldNetworkRxRate], "%d", &status.NetworkRxRate)
	fmt.Sscanf(values[FieldNetworkTxRate], "%d", &status.NetworkTxRate)

	// Int64 field
	fmt.Sscanf(values[FieldUptimeSeconds], "%d", &status.UptimeSeconds)

	// Int fields
	fmt.Sscanf(values[FieldTCPConnections], "%d", &status.TCPConnections)
	fmt.Sscanf(values[FieldUDPConnections], "%d", &status.UDPConnections)

	// String fields
	status.PublicIPv4 = values[FieldPublicIPv4]
	status.PublicIPv6 = values[FieldPublicIPv6]
	status.AgentVersion = values[FieldAgentVersion]
	status.Platform = values[FieldPlatform]
	status.Arch = values[FieldArch]

	// CPU details
	fmt.Sscanf(values[FieldCPUCores], "%d", &status.CPUCores)
	status.CPUModelName = values[FieldCPUModelName]
	fmt.Sscanf(values[FieldCPUMHz], "%f", &status.CPUMHz)

	// Swap memory
	fmt.Sscanf(values[FieldSwapTotal], "%d", &status.SwapTotal)
	fmt.Sscanf(values[FieldSwapUsed], "%d", &status.SwapUsed)
	fmt.Sscanf(values[FieldSwapPercent], "%f", &status.SwapPercent)

	// Disk I/O
	fmt.Sscanf(values[FieldDiskReadBytes], "%d", &status.DiskReadBytes)
	fmt.Sscanf(values[FieldDiskWriteBytes], "%d", &status.DiskWriteBytes)
	fmt.Sscanf(values[FieldDiskReadRate], "%d", &status.DiskReadRate)
	fmt.Sscanf(values[FieldDiskWriteRate], "%d", &status.DiskWriteRate)
	fmt.Sscanf(values[FieldDiskIOPS], "%d", &status.DiskIOPS)

	// Pressure Stall Information (PSI)
	fmt.Sscanf(values[FieldPSICPUSome], "%f", &status.PSICPUSome)
	fmt.Sscanf(values[FieldPSICPUFull], "%f", &status.PSICPUFull)
	fmt.Sscanf(values[FieldPSIMemorySome], "%f", &status.PSIMemorySome)
	fmt.Sscanf(values[FieldPSIMemoryFull], "%f", &status.PSIMemoryFull)
	fmt.Sscanf(values[FieldPSIIOSome], "%f", &status.PSIIOSome)
	fmt.Sscanf(values[FieldPSIIOFull], "%f", &status.PSIIOFull)

	// Network extended stats
	fmt.Sscanf(values[FieldNetworkRxPackets], "%d", &status.NetworkRxPackets)
	fmt.Sscanf(values[FieldNetworkTxPackets], "%d", &status.NetworkTxPackets)
	fmt.Sscanf(values[FieldNetworkRxErrors], "%d", &status.NetworkRxErrors)
	fmt.Sscanf(values[FieldNetworkTxErrors], "%d", &status.NetworkTxErrors)
	fmt.Sscanf(values[FieldNetworkRxDropped], "%d", &status.NetworkRxDropped)
	fmt.Sscanf(values[FieldNetworkTxDropped], "%d", &status.NetworkTxDropped)

	// Socket statistics
	fmt.Sscanf(values[FieldSocketsUsed], "%d", &status.SocketsUsed)
	fmt.Sscanf(values[FieldSocketsTCPInUse], "%d", &status.SocketsTCPInUse)
	fmt.Sscanf(values[FieldSocketsUDPInUse], "%d", &status.SocketsUDPInUse)
	fmt.Sscanf(values[FieldSocketsTCPOrphan], "%d", &status.SocketsTCPOrphan)
	fmt.Sscanf(values[FieldSocketsTCPTW], "%d", &status.SocketsTCPTW)

	// Process statistics
	fmt.Sscanf(values[FieldProcessesTotal], "%d", &status.ProcessesTotal)
	fmt.Sscanf(values[FieldProcessesRunning], "%d", &status.ProcessesRunning)
	fmt.Sscanf(values[FieldProcessesBlocked], "%d", &status.ProcessesBlocked)

	// File descriptors
	fmt.Sscanf(values[FieldFileNrAllocated], "%d", &status.FileNrAllocated)
	fmt.Sscanf(values[FieldFileNrMax], "%d", &status.FileNrMax)

	// Context switches and interrupts
	fmt.Sscanf(values[FieldContextSwitches], "%d", &status.ContextSwitches)
	fmt.Sscanf(values[FieldInterrupts], "%d", &status.Interrupts)

	// Kernel info
	status.KernelVersion = values[FieldKernelVersion]
	status.Hostname = values[FieldHostname]

	// Virtual memory statistics
	fmt.Sscanf(values[FieldVMPageIn], "%d", &status.VMPageIn)
	fmt.Sscanf(values[FieldVMPageOut], "%d", &status.VMPageOut)
	fmt.Sscanf(values[FieldVMSwapIn], "%d", &status.VMSwapIn)
	fmt.Sscanf(values[FieldVMSwapOut], "%d", &status.VMSwapOut)
	fmt.Sscanf(values[FieldVMOOMKill], "%d", &status.VMOOMKill)

	// Entropy pool
	fmt.Sscanf(values[FieldEntropyAvailable], "%d", &status.EntropyAvailable)

	return status
}

// ToRedisFields converts SystemStatus to Redis hash fields.
func ToRedisFields(status *commondto.SystemStatus) map[string]interface{} {
	return map[string]interface{}{
		FieldCPUPercent:    fmt.Sprintf("%.2f", status.CPUPercent),
		FieldMemoryPercent: fmt.Sprintf("%.2f", status.MemoryPercent),
		FieldMemoryUsed:    fmt.Sprintf("%d", status.MemoryUsed),
		FieldMemoryTotal:   fmt.Sprintf("%d", status.MemoryTotal),
		FieldMemoryAvail:   fmt.Sprintf("%d", status.MemoryAvail),
		FieldDiskPercent:   fmt.Sprintf("%.2f", status.DiskPercent),
		FieldDiskUsed:      fmt.Sprintf("%d", status.DiskUsed),
		FieldDiskTotal:     fmt.Sprintf("%d", status.DiskTotal),
		FieldUptimeSeconds: fmt.Sprintf("%d", status.UptimeSeconds),

		FieldLoadAvg1:  fmt.Sprintf("%.2f", status.LoadAvg1),
		FieldLoadAvg5:  fmt.Sprintf("%.2f", status.LoadAvg5),
		FieldLoadAvg15: fmt.Sprintf("%.2f", status.LoadAvg15),

		FieldNetworkRxBytes: fmt.Sprintf("%d", status.NetworkRxBytes),
		FieldNetworkTxBytes: fmt.Sprintf("%d", status.NetworkTxBytes),
		FieldNetworkRxRate:  fmt.Sprintf("%d", status.NetworkRxRate),
		FieldNetworkTxRate:  fmt.Sprintf("%d", status.NetworkTxRate),

		FieldTCPConnections: fmt.Sprintf("%d", status.TCPConnections),
		FieldUDPConnections: fmt.Sprintf("%d", status.UDPConnections),

		// Truncate string fields to prevent memory exhaustion
		FieldPublicIPv4: truncateString(status.PublicIPv4, MaxIPLength),
		FieldPublicIPv6: truncateString(status.PublicIPv6, MaxIPLength),

		FieldAgentVersion: truncateString(status.AgentVersion, MaxVersionLength),
		FieldPlatform:     truncateString(status.Platform, MaxVersionLength),
		FieldArch:         truncateString(status.Arch, MaxVersionLength),

		// CPU details
		FieldCPUCores:     fmt.Sprintf("%d", status.CPUCores),
		FieldCPUModelName: truncateString(status.CPUModelName, MaxCPUModelNameLength),
		FieldCPUMHz:       fmt.Sprintf("%.2f", status.CPUMHz),

		// Swap memory
		FieldSwapTotal:   fmt.Sprintf("%d", status.SwapTotal),
		FieldSwapUsed:    fmt.Sprintf("%d", status.SwapUsed),
		FieldSwapPercent: fmt.Sprintf("%.2f", status.SwapPercent),

		// Disk I/O
		FieldDiskReadBytes:  fmt.Sprintf("%d", status.DiskReadBytes),
		FieldDiskWriteBytes: fmt.Sprintf("%d", status.DiskWriteBytes),
		FieldDiskReadRate:   fmt.Sprintf("%d", status.DiskReadRate),
		FieldDiskWriteRate:  fmt.Sprintf("%d", status.DiskWriteRate),
		FieldDiskIOPS:       fmt.Sprintf("%d", status.DiskIOPS),

		// Pressure Stall Information (PSI)
		FieldPSICPUSome:    fmt.Sprintf("%.2f", status.PSICPUSome),
		FieldPSICPUFull:    fmt.Sprintf("%.2f", status.PSICPUFull),
		FieldPSIMemorySome: fmt.Sprintf("%.2f", status.PSIMemorySome),
		FieldPSIMemoryFull: fmt.Sprintf("%.2f", status.PSIMemoryFull),
		FieldPSIIOSome:     fmt.Sprintf("%.2f", status.PSIIOSome),
		FieldPSIIOFull:     fmt.Sprintf("%.2f", status.PSIIOFull),

		// Network extended stats
		FieldNetworkRxPackets: fmt.Sprintf("%d", status.NetworkRxPackets),
		FieldNetworkTxPackets: fmt.Sprintf("%d", status.NetworkTxPackets),
		FieldNetworkRxErrors:  fmt.Sprintf("%d", status.NetworkRxErrors),
		FieldNetworkTxErrors:  fmt.Sprintf("%d", status.NetworkTxErrors),
		FieldNetworkRxDropped: fmt.Sprintf("%d", status.NetworkRxDropped),
		FieldNetworkTxDropped: fmt.Sprintf("%d", status.NetworkTxDropped),

		// Socket statistics
		FieldSocketsUsed:      fmt.Sprintf("%d", status.SocketsUsed),
		FieldSocketsTCPInUse:  fmt.Sprintf("%d", status.SocketsTCPInUse),
		FieldSocketsUDPInUse:  fmt.Sprintf("%d", status.SocketsUDPInUse),
		FieldSocketsTCPOrphan: fmt.Sprintf("%d", status.SocketsTCPOrphan),
		FieldSocketsTCPTW:     fmt.Sprintf("%d", status.SocketsTCPTW),

		// Process statistics
		FieldProcessesTotal:   fmt.Sprintf("%d", status.ProcessesTotal),
		FieldProcessesRunning: fmt.Sprintf("%d", status.ProcessesRunning),
		FieldProcessesBlocked: fmt.Sprintf("%d", status.ProcessesBlocked),

		// File descriptors
		FieldFileNrAllocated: fmt.Sprintf("%d", status.FileNrAllocated),
		FieldFileNrMax:       fmt.Sprintf("%d", status.FileNrMax),

		// Context switches and interrupts
		FieldContextSwitches: fmt.Sprintf("%d", status.ContextSwitches),
		FieldInterrupts:      fmt.Sprintf("%d", status.Interrupts),

		// Kernel info (truncated to prevent memory exhaustion)
		FieldKernelVersion: truncateString(status.KernelVersion, MaxKernelVersionLen),
		FieldHostname:      truncateString(status.Hostname, MaxHostnameLength),

		// Virtual memory statistics
		FieldVMPageIn:  fmt.Sprintf("%d", status.VMPageIn),
		FieldVMPageOut: fmt.Sprintf("%d", status.VMPageOut),
		FieldVMSwapIn:  fmt.Sprintf("%d", status.VMSwapIn),
		FieldVMSwapOut: fmt.Sprintf("%d", status.VMSwapOut),
		FieldVMOOMKill: fmt.Sprintf("%d", status.VMOOMKill),

		// Entropy pool
		FieldEntropyAvailable: fmt.Sprintf("%d", status.EntropyAvailable),
	}
}
