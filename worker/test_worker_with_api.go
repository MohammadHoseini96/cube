package worker

import (
	"cube/store"
	"fmt"
	"github.com/golang-collections/collections/queue"
	"strconv"
	"strings"
)

func ServeWorkerWithApi(host string, port int) {
	//host := os.Getenv("CUBE_HOST")
	//port, _ := strconv.Atoi(os.Getenv("CUBE_PORT"))

	fmt.Println("Starting Cube worker")

	w := Worker{
		Queue: *queue.New(),
		Db:    store.NewInMemoryTaskStore(),
	}
	api := Api{
		Address: host,
		Port:    port,
		Worker:  &w,
	}

	go w.RunTasks()
	go w.CollectStats()
	go w.UpdateTasks()
	api.Start()
}

func ServeWorkersWithApi(hostPortMap map[int]string, dbType string) {
	idx := 1
	for port, host := range hostPortMap {
		//w := Worker{
		//	Queue: *queue.New(),
		//	Db:    store.NewInMemoryTaskStore(),
		//}
		w := New(fmt.Sprintf("worker-%d", idx), dbType)
		api := Api{
			Address: host,
			Port:    port,
			Worker:  w,
		}

		go w.RunTasks()
		go w.CollectStats()
		go w.UpdateTasks()
		go api.Start()
		idx++
	}
}

func ServeWorkersByAddressWithApi(workers []string, dbType string) {
	for idx, address := range workers {
		//w := Worker{
		//	Queue: *queue.New(),
		//	Db:    store.NewInMemoryTaskStore(),
		//}
		addr := strings.Split(address, ":")
		host := addr[0]
		port, _ := strconv.Atoi(addr[1])
		w := New(fmt.Sprintf("worker-%d", idx+1), dbType)
		api := Api{
			Address: host,
			Port:    port,
			Worker:  w,
		}

		go w.RunTasks()
		go w.CollectStats()
		go w.UpdateTasks()
		go api.Start()
	}
}

//func runTask(w *Worker) {
//	for {
//		if w.Queue.Len() != 0 {
//			result := w.RunTask()
//			if result.Error != nil {
//				log.Printf("Error running task: %v\n", result.Error)
//			}
//		} else {
//			log.Println("No tasks to process currently.")
//		}
//		log.Println("Sleeping for 6sec.")
//		time.Sleep(10 * time.Second)
//	}
//}
