package worker

import (
	"cube/task"
	"fmt"
	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
)

func ServeWorkerWithApi(host string, port int) {
	//host := os.Getenv("CUBE_HOST")
	//port, _ := strconv.Atoi(os.Getenv("CUBE_PORT"))

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

	go w.RunTasks()
	go w.CollectStats()
	api.Start()
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
