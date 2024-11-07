package main

import "cube/manager"

//TIP To run your code, right-click the code and select <b>Run</b>. Alternatively, click
// the <icon src="AllIcons.Actions.Execute"/> icon in the gutter and select the <b>Run</b> menu item from here.

func main() {

	//fmt.Printf("create a test container\n")
	//dockerTask, createResult := task.CreateContainer()
	//
	//time.Sleep(time.Second * 5)
	//
	//fmt.Printf("stopping container %s\n", createResult.ContainerId)
	//_ = task.StopContainer(dockerTask, createResult.ContainerId)

	//worker.Test_worker_with_state_transition()

	//host := os.Getenv("CUBE_HOST")
	//port, _ := strconv.Atoi(os.Getenv("CUBE_PORT"))
	//host := "localhost"
	//port := 5555
	//numberOFRandomTasks := 3
	//worker.ServeWorkerWithApi(host, port)
	//manager.ServeManager(host, port, numberOFRandomTasks)

	//whost := os.Getenv("CUBE_WORKER_HOST")
	//wport, _ := strconv.Atoi(os.Getenv("CUBE_WORKER_PORT"))
	//
	//mhost := os.Getenv("CUBE_MANAGER_HOST")
	//mport, _ := strconv.Atoi(os.Getenv("CUBE_MANAGER_PORT"))

	managerHost := "localhost"
	managerPort := 5555
	workerHost := "localhost"
	workerPort := 5556
	workerPorts := make(map[int]string)
	workerPorts[workerPort] = workerHost
	workerPorts[workerPort+1] = workerHost
	workerPorts[workerPort+2] = workerHost
	manager.ServeManagerWithMultipleWorkers(managerHost, managerPort, workerPorts)

}

//TIP See GoLand help at <a href="https://www.jetbrains.com/help/go/">jetbrains.com/help/go/</a>.
// Also, you can try interactive lessons for GoLand by selecting 'Help | Learn IDE Features' from the main menu.
