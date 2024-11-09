package scheduler

import (
	"cube/node"
	"cube/task"
	"log"
	"math"
	"time"
)

type Scheduler interface {
	SelectCandidateNodes(t task.Task, nodes []*node.Node) []*node.Node
	Score(t task.Task, nodes []*node.Node) map[string]float64
	Pick(scores map[string]float64, candidates []*node.Node) *node.Node
}

type RoundRobin struct {
	Name       string
	LastWorker int
}

func (r *RoundRobin) SelectCandidateNodes(task task.Task, nodes []*node.Node) []*node.Node {
	return nodes
}

func (r *RoundRobin) Score(task task.Task, nodes []*node.Node) map[string]float64 {
	nodeScores := make(map[string]float64)

	var newWorker int
	if r.LastWorker+1 < len(nodes) {
		newWorker = r.LastWorker + 1
		r.LastWorker++
	} else {
		newWorker = 0
		r.LastWorker = 0
	}

	for idx, n := range nodes {
		if idx == newWorker {
			nodeScores[n.Name] = 0.1
		} else {
			nodeScores[n.Name] = 1.0
		}
	}

	return nodeScores
}

func (r *RoundRobin) Pick(scores map[string]float64, candidates []*node.Node) *node.Node {
	var bestNode *node.Node
	var lowestScore float64

	for idx, n := range candidates {
		if idx == 0 {
			bestNode = n
			lowestScore = scores[n.Name]
			continue
		}

		if lowestScore < scores[n.Name] {
			lowestScore = scores[n.Name]
			bestNode = n
		}
	}

	return bestNode
}

type Epvm struct {
	Name string
}

func (e *Epvm) SelectCandidateNodes(t task.Task, nodes []*node.Node) []*node.Node {
	var candidates []*node.Node
	for n := range nodes {
		if checkDisk(t, nodes[n].Disk-nodes[n].DiskAllocated) {
			candidates = append(candidates, nodes[n])
		}
	}

	return candidates
}

func checkDisk(t task.Task, diskAvailable int64) bool {
	return t.Disk <= diskAvailable
}

const (
	// LIEB square ice constant
	// https://en.wikipedia.org/wiki/Lieb%27s_square_ice_constant
	LIEB = 1.53960071783900203869
)

func (e *Epvm) Score(t task.Task, nodes []*node.Node) map[string]float64 {
	nodeScores := make(map[string]float64)
	maxJobs := 4.0

	for _, n := range nodes {
		cpuUsage, err := calculateCpuUsage(n)
		if err != nil {
			log.Printf("error calculating CPU usage for node %s, skipping: %v\n", n.Name, err)
			continue
		}
		cpuLoad := calculateLoad(*cpuUsage, math.Pow(2, 0.8))

		memoryAllocated := float64(n.Stats.MemUsedKb()) + float64(n.MemoryAllocated)
		memoryPercentAllocated := memoryAllocated / float64(n.Memory)

		newMemPercent := calculateLoad(memoryAllocated+float64(t.Memory/1000), float64(n.Memory))

		memCost :=
			math.Pow(LIEB, newMemPercent) +
				math.Pow(LIEB, (float64(n.TaskCount+1))/maxJobs) -
				math.Pow(LIEB, memoryPercentAllocated) -
				math.Pow(LIEB, float64(n.TaskCount)/float64(maxJobs))

		cpuCost :=
			math.Pow(LIEB, cpuLoad) +
				math.Pow(LIEB, (float64(n.TaskCount+1))/maxJobs) -
				math.Pow(LIEB, cpuLoad) -
				math.Pow(LIEB, float64(n.TaskCount)/float64(maxJobs))

		nodeScores[n.Name] = memCost + cpuCost
	}

	return nodeScores
}

func (e *Epvm) Pick(scores map[string]float64, candidates []*node.Node) *node.Node {
	minCost := 0.0
	var bestNode *node.Node
	for idx, n := range candidates {
		if idx == 0 {
			minCost = scores[n.Name]
			bestNode = n
			continue
		}

		if scores[n.Name] < minCost {
			minCost = scores[n.Name]
			bestNode = n
		}
	}

	return bestNode
}

func calculateLoad(usage float64, capacity float64) float64 {
	return usage / capacity
}

func calculateCpuUsage(n *node.Node) (*float64, error) {
	//stat1 := getNodeStats(n)
	stat1, err := n.GetStats()
	if err != nil {
		return nil, err
	}
	time.Sleep(3 * time.Second)
	//stat2 := getNodeStats(n)
	stat2, err := n.GetStats()
	if err != nil {
		return nil, err
	}

	stat1Idle := stat1.CpuStats.Idle + stat1.CpuStats.Iowait
	stat2Idle := stat2.CpuStats.Idle + stat2.CpuStats.Iowait

	stat1NonIdle :=
		stat1.CpuStats.User +
			stat1.CpuStats.Nice +
			stat1.CpuStats.System +
			stat1.CpuStats.Irq +
			stat1.CpuStats.Softirq +
			stat1.CpuStats.Steal
	stat2NonIdle :=
		stat2.CpuStats.User +
			stat2.CpuStats.Nice +
			stat2.CpuStats.System +
			stat2.CpuStats.Irq +
			stat2.CpuStats.Softirq +
			stat2.CpuStats.Steal

	stat1Total := stat1Idle + stat1NonIdle
	stat2Total := stat2Idle + stat2NonIdle

	total := stat2Total - stat1Total
	idle := stat2Idle - stat1Idle

	var cpuPercentUsage float64
	if total == 0 && idle == 0 {
		cpuPercentUsage = 0.00
	} else {
		cpuPercentUsage = (float64(total) - float64(idle)) / float64(total)
	}
	return &cpuPercentUsage, nil
}
