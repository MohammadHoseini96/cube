package node

import (
	"cube/stats"
	"cube/utils"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
)

type Node struct {
	Name            string
	Ip              string
	Api             string
	Cores           int64
	Memory          int64
	MemoryAllocated int64
	Disk            int64
	DiskAllocated   int64
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

// GetStats TODO: TaskCount needs to be updated, so we can show it when using node command from the cmd package.
// Stats.TaskCount also requires such change.
func (n *Node) GetStats() (*stats.Stats, error) {
	var resp *http.Response
	var err error

	url := fmt.Sprintf("%s/stats", n.Api)
	resp, err = utils.HTTPWithRetry(http.Get, url)
	if err != nil {
		msg := fmt.Sprintf("Unable to connect to %v. Permanent failure.\n", n.Api)
		log.Println(msg)
		return nil, errors.New(msg)
	}

	if resp.StatusCode != 200 {
		msg := fmt.Sprintf("Error retrieving stats from %v: %v", n.Api, err)
		log.Println(msg)
		return nil, errors.New(msg)
	}

	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var s stats.Stats
	err = json.Unmarshal(body, &s)
	if err != nil {
		msg := fmt.Sprintf("error decoding message while getting stats for node %s", n.Name)
		log.Println(msg)
		return nil, errors.New(msg)
	}

	if s.MemStats == nil || s.DiskStats == nil {
		return nil, fmt.Errorf("error getting stats from node %s", n.Name)
	}

	n.Memory = int64(s.MemTotalKb())
	n.Disk = int64(s.DiskTotal())
	n.Stats = s

	return &n.Stats, nil
}

func (n *Node) GetCpuUsage() (float64, error) {
	var resp *http.Response
	var err error
	interval := 3

	url := fmt.Sprintf("%s/stats/cpu-usage/%d", n.Api, interval)
	resp, err = utils.HTTPWithRetry(http.Get, url)
	if err != nil {
		msg := fmt.Sprintf("Unable to connect to %v. Permanent failure.\n", n.Api)
		log.Println(msg)
		return 0, errors.New(msg)
	}

	if resp.StatusCode != 200 {
		msg := fmt.Sprintf("Error retrieving cpu usage from %v: %v", n.Api, err)
		log.Println(msg)
		return 0, errors.New(msg)
	}

	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var u stats.CPUUsage
	err = json.Unmarshal(body, &u)
	if err != nil {
		msg := fmt.Sprintf("error decoding message while getting cpu usage for node %s", n.Name)
		log.Println(msg)
		return 0, errors.New(msg)
	}

	if u.Percentage == 0 {
		return 0, fmt.Errorf("error getting cpu usage from node %s", n.Name)
	}

	return u.Percentage, nil
}
