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

func (n *Node) GetStats() (*stats.Stats, error) {
	var resp *http.Response
	var err error

	url := fmt.Sprintf("%s/s", n.Api)
	resp, err = utils.HTTPWithRetry(http.Get, url)
	if err != nil {
		msg := fmt.Sprintf("Unable to connect to %v. Permanent failure.\n", n.Api)
		log.Println(msg)
		return nil, errors.New(msg)
	}

	if resp.StatusCode != 200 {
		msg := fmt.Sprintf("Error retrieving s from %v: %v", n.Api, err)
		log.Println(msg)
		return nil, errors.New(msg)
	}

	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var s stats.Stats
	err = json.Unmarshal(body, &s)
	if err != nil {
		msg := fmt.Sprintf("error decoding message while getting s for node %s", n.Name)
		log.Println(msg)
		return nil, errors.New(msg)
	}

	if s.MemStats == nil || s.DiskStats == nil {
		return nil, fmt.Errorf("error getting s from node %s", n.Name)
	}

	n.Memory = int64(s.MemTotalKb())
	n.Disk = int64(s.DiskTotal())
	n.Stats = s

	return &n.Stats, nil
}
