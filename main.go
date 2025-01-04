package main

import (
	"bufio"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"time"
)

type (
	EventData struct {
		Message string `json:"message,omitempty"`
	}
)

var eventChannel chan *EventData

func main() {
	eventChannel = make(chan *EventData)

	go fileTail("somefile.txt")
	http.HandleFunc("/events", eventsHandler)
	http.HandleFunc("/", showMain)

	http.ListenAndServe(":8080", nil)
}

func eventsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Expose-Headers", "Content-Type")

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case t := <-ticker.C:
			_ = t

			dt := <-eventChannel

			divCard := "<div class=\"event-message-div\">" + dt.Message + "</div>"

			fmt.Fprintf(w, "data: %s\n\n", divCard)
			fmt.Printf("data: %v\n", divCard)

			// Flush the response to ensure the event is sent
			flusher, ok := w.(http.Flusher)
			if ok {
				flusher.Flush()
			}
		case <-r.Context().Done():
			log.Println("Client disconnected")
			return
		}
	}

}

func fileTail(filepath string) {
	file, err := os.Open(filepath)
	if err != nil {
		fmt.Println("Failed to open file: ", filepath)
		return
	}
	defer file.Close()

	// Read last n lines
	lines, err := readLastNLines(filepath, 5)
	if err != nil {
		panic(err.Error())
	}

	for _, line := range lines {
		fmt.Printf("Line is: %v\n", line)
		ed1 := &EventData{Message: line}
		eventChannel <- ed1
	}

	// Watch new lines
	offset, _ := file.Seek(0, io.SeekEnd)
	buffer := make([]byte, 1024)
	for {
		readBytes, err := file.ReadAt(buffer, offset)
		if err != nil {
			if err != io.EOF {
				fmt.Println("Error reading lines:", err)
				break
			}
		}
		offset += int64(readBytes)
		if readBytes != 0 {
			s := string(buffer[:readBytes])

			ed := &EventData{Message: s}
			eventChannel <- ed
			fmt.Printf("%s", s)
		}
		time.Sleep(100 * time.Microsecond)
	}
}

func readLastNLines(filePath string, n int) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) > n {
			lines = lines[1:]
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}

func showMain(w http.ResponseWriter, r *http.Request) {
	tmpl := getTemplate("index.html")
	tmpl.Execute(w, "test")
}

func getTemplate(tempName string) *template.Template {
	fp := path.Join("static", tempName)
	tmpl, err := template.ParseFiles(fp)
	if err != nil {
		panic("template not found")
	}
	return tmpl
}
