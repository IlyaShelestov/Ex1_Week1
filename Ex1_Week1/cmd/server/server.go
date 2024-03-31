package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	CONN_PORT = ":9090"
	CONN_TYPE = "tcp"
)

var (
	clients       = make(map[net.Conn]string)
	addr          = make(map[net.Conn]string)
	mutex         sync.Mutex
	historyLog    = "history.log"
	tasks         = make(map[string]Task)
	taskIDCounter int
)

type Task struct {
	ID          string
	Description string
	Owner       string
}

func handleConnection(conn net.Conn) {
	nickname := "Anonymous"

	mutex.Lock()
	clients[conn] = nickname
	addr[conn] = conn.RemoteAddr().String()
	mutex.Unlock()

	defer func() {
		mutex.Lock()
		delete(clients, conn)
		delete(addr, conn)
		mutex.Unlock()
		conn.Close()
	}()

	logFile, err := os.OpenFile(historyLog, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println("Error opening history log file:", err)
		return
	}
	defer logFile.Close()

	log.Printf("Client %s (%s) connected.", addr[conn], nickname)

	for {
		netData, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			log.Printf("Client %s (%s) disconnected.", addr[conn], nickname)
			broadcastMessage(fmt.Sprintf("%s disconnected from the chat!\n", nickname), conn)
			break
		}

		handleCommands(conn, &nickname, strings.TrimSpace(string(netData)), logFile)
	}
}

func handleCommands(conn net.Conn, nickname *string, message string, logFile *os.File) {
	if strings.HasPrefix(message, "/quit") {
		conn.Write([]byte("Goodbye!\n"))
		conn.Close()
	} else if strings.HasPrefix(message, "/history") {
		sendHistory(conn)
	} else if strings.HasPrefix(message, "/nickname") {
		parts := strings.SplitN(message, " ", 2)
		if len(parts) == 2 {
			changeNickname(conn, nickname, parts[1])
		}
	} else if strings.HasPrefix(message, "/users") {
		sendUsersList(conn)
	} else if strings.HasPrefix(message, "/task add") {
		parts := strings.SplitN(message, " ", 3)
		if len(parts) == 3 {
			addTask(conn, *nickname, parts[2])
		}
	} else if strings.HasPrefix(message, "/task list") {
		listTasks(conn)
	} else if strings.HasPrefix(message, "/task delete") {
		parts := strings.SplitN(message, " ", 3)
		if len(parts) == 3 {
			deleteTask(conn, parts[2])
		}
	} else {
		logMessage(*nickname, message, logFile)
		response := fmt.Sprintf("%s: %s\n", *nickname, message)
		broadcastMessage(response, conn)
	}
}

func sendUsersList(conn net.Conn) {
	mutex.Lock()
	defer mutex.Unlock()

	var users []string
	for _, nickname := range clients {
		users = append(users, nickname)
	}

	usersList := strings.Join(users, ", ")
	message := fmt.Sprintf("Connected users: %s\n", usersList)
	conn.Write([]byte(message))
}

func changeNickname(conn net.Conn, nickname *string, newNickname string) {
	oldNickname := *nickname
	*nickname = newNickname

	mutex.Lock()
	clients[conn] = newNickname
	mutex.Unlock()

	conn.Write([]byte(fmt.Sprintf("Nickname changed to %s\n", newNickname)))

	log.Printf("Client %s (%s) changed nickname to %s.", addr[conn], oldNickname, newNickname)
	broadcastMessage(fmt.Sprintf("'%s' changed nickname to '%s'\n", oldNickname, newNickname), conn)
}

func sendHistory(conn net.Conn) {
	file, err := os.Open(historyLog)
	if err != nil {
		conn.Write([]byte("Error reading history.\n"))
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		message := scanner.Text() + "\n"
		_, err := conn.Write([]byte(message))
		if err != nil {
			log.Printf("Error sending history to client: %s", err)
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading from history file: %s", err)
		conn.Write([]byte("Error occurred while reading history.\n"))
	}
}

func logMessage(nickname string, message string, logFile *os.File) {
	currentTime := time.Now().Format(time.RFC1123)
	logEntry := fmt.Sprintf("%s: %s - %s\n", currentTime, nickname, message)
	logFile.WriteString(logEntry)
}

func broadcastMessage(message string, sender net.Conn) {
	mutex.Lock()
	defer mutex.Unlock()
	for conn := range clients {
		if conn != sender {
			conn.Write([]byte(message))
		}
	}
}

func addTask(conn net.Conn, owner, description string) {
	mutex.Lock()
	defer mutex.Unlock()

	taskIDCounter++
	taskID := fmt.Sprintf("%d", taskIDCounter)
	tasks[taskID] = Task{ID: taskID, Description: description, Owner: owner}

	conn.Write([]byte(fmt.Sprintf("Task added with ID %s\n", taskID)))
}

func listTasks(conn net.Conn) {
	mutex.Lock()
	defer mutex.Unlock()

	var taskDescriptions []string
	for _, task := range tasks {
		taskDescriptions = append(taskDescriptions, fmt.Sprintf("ID: %s, Owner: %s, Description: %s", task.ID, task.Owner, task.Description))
	}

	if len(taskDescriptions) == 0 {
		conn.Write([]byte("No tasks found.\n"))
	} else {
		conn.Write([]byte(strings.Join(taskDescriptions, "; ") + "\n"))
	}
}

func deleteTask(conn net.Conn, taskID string) {
	mutex.Lock()
	defer mutex.Unlock()

	if _, ok := tasks[taskID]; ok {
		delete(tasks, taskID)
		conn.Write([]byte("Task deleted successfully.\n"))
	} else {
		conn.Write([]byte("Task not found.\n"))
	}
}

func main() {
	listener, err := net.Listen(CONN_TYPE, CONN_PORT)
	if err != nil {
		log.Fatal("Error starting TCP server:", err)
	}
	defer listener.Close()
	log.Println("Server listening on", CONN_PORT)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Error accepting connection:", err)
			continue
		}
		go handleConnection(conn)
	}
}
