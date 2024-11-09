package stats

import (
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/load"
	"github.com/shirou/gopsutil/mem"
	"log"
)

type Stats struct {
	MemStats     *mem.VirtualMemoryStat
	DiskStats    *disk.UsageStat
	CpuStats     *cpu.TimesStat
	LoadStats    *load.MiscStat
	AvgLoadStats *load.AvgStat
	TaskCount    int
}

func (s *Stats) MemTotalKb() uint64 {
	return s.MemStats.Total
}

func (s *Stats) MemAvailableKb() uint64 {
	return s.MemStats.Available
}

func (s *Stats) MemUsedKb() uint64 {
	return s.MemStats.Used
}

func (s *Stats) MemUsedPercent() float64 {
	return s.MemStats.UsedPercent
}

func (s *Stats) DiskTotal() uint64 {
	return s.DiskStats.Total
}

func (s *Stats) DiskFree() uint64 {
	return s.DiskStats.Free
}

func (s *Stats) DiskUsed() uint64 {
	return s.DiskStats.Used
}

func (s *Stats) CpuUsage() float64 {
	/*
		based off the solution's algorithm:
		https://stackoverflow.com/questions/23367857/accurate-calculation-of-cpu-usage-given-in-percentage-in-linux
	*/
	idle := s.CpuStats.Idle + s.CpuStats.Iowait
	nonIdle :=
		s.CpuStats.User +
			s.CpuStats.Irq +
			s.CpuStats.Nice +
			s.CpuStats.Steal +
			s.CpuStats.System +
			s.CpuStats.Softirq
	total := idle + nonIdle

	if total == 0 {
		return float64(0)
	}

	return (total - idle) / total
}

func GetStats() *Stats {
	return &Stats{
		MemStats:     GetMemoryInfo(),
		DiskStats:    GetDiskInfo(),
		CpuStats:     GetCpuInfo(),
		LoadStats:    GetLoadInfo(),
		AvgLoadStats: GetLoadAvgInfo(),
	}
}

func GetMemoryInfo() *mem.VirtualMemoryStat {
	memStats, err := mem.VirtualMemory()
	if err != nil {
		log.Printf("Error reading memory info : %v\n", err)
		return &mem.VirtualMemoryStat{}
	}

	return memStats
}

func GetDiskInfo() *disk.UsageStat {
	diskStats, err := disk.Usage("/")
	if err != nil {
		log.Printf("Error reading disk from / : %v\n", err)
		return &disk.UsageStat{}
	}

	return diskStats
}

func GetCpuInfo() *cpu.TimesStat {
	cpuStats, err := cpu.Times(false) // 'false' returns aggregated data across all CPUs
	if err != nil {
		log.Printf("Error reading cpu times : %v\n", err)
		return &cpu.TimesStat{}
	}

	return &cpuStats[0]
}

func GetLoadInfo() *load.MiscStat {
	loadStats, err := load.Misc()
	if err != nil {
		log.Printf("Error reading load misc info : %v\n", err)
		return &load.MiscStat{}
	}

	return loadStats
}

func GetLoadAvgInfo() *load.AvgStat {
	// Avg method for Windows may return 0 values!
	loadAvg, err := load.Avg()
	if err != nil {
		log.Printf("Error reading load average info : %v\n", err)
		return &load.AvgStat{}
	}

	return loadAvg
}
