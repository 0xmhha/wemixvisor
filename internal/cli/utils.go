package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"time"
)

// tailLogs displays the last n lines of a log file
func tailLogs(logFile string, lines int) error {
	file, err := os.Open(logFile)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	// Read all lines into a buffer
	var buffer []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		buffer = append(buffer, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read log file: %w", err)
	}

	// Display the last n lines
	start := len(buffer) - lines
	if start < 0 {
		start = 0
	}

	for i := start; i < len(buffer); i++ {
		fmt.Println(buffer[i])
	}

	return nil
}

// followLogs continuously displays new log entries
func followLogs(logFile string) error {
	file, err := os.Open(logFile)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	// Start at the end of the file
	file.Seek(0, io.SeekEnd)

	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				// Wait for new data
				time.Sleep(100 * time.Millisecond)
				continue
			}
			return fmt.Errorf("failed to read log file: %w", err)
		}
		fmt.Print(line)
	}
}