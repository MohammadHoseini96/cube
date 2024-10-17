package worker

import (
	"cube/task"
	"fmt"
	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
	"log"
	"time"
)

func Serve_worker_with_api() {
	//host := os.Getenv("CUBE_HOST")
	//port, _ := strconv.Atoi(os.Getenv("CUBE_PORT"))

	host := "127.0.0.1"
	port := 8080

	fmt.Println("Starting Cube worker")

	w := Worker{
		Queue: *queue.New(),
		Db:    make(map[uuid.UUID]*task.Task),
	}
	api := Api{
		Address: host,
		Port:    port,
		Worker:  &w,
	}

	go runTask(&w)
	api.Start()
}

func runTask(w *Worker) {
	for {
		if w.Queue.Len() != 0 {
			result := w.RunTask()
			if result.Error != nil {
				log.Printf("Error running task: %v\n", result.Error)
			}
		} else {
			log.Println("No tasks to process currently.")
		}
		log.Println("Sleeping for 6sec.")
		time.Sleep(6 * time.Second)
	}
}
