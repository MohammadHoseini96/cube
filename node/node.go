package node

import "google.golang.org/grpc/benchmark/stats"

type Node struct {
	Name            string
	Ip              string
	Api             string
	Cores           int
	Memory          int
	MemoryAllocated int
	Disk            int
	DiskAllocated   int
	Stats           stats.Stats
	Role            string
	TaskCount       int
}

func NewNode(name string, api string, role string) *Node {
	return &Node{
		Name: name,
		Api:  api,
		Role: role,
	}
}
