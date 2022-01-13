package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type Cursor struct {}

func (c *Cursor) hide() {
	fmt.Printf("\033[?25l")
}

func (c *Cursor) show()  {
	fmt.Printf("\033[?25h")
}

func (c *Cursor) moveUp(rows int) {
	fmt.Printf("\033[%dA", rows)
}

func (c *Cursor) moveDown(rows int) {
	fmt.Printf("\033[%dB", rows)
}

func (c *Cursor) clearLine()  {
	fmt.Printf("\033[2K")
}
type pushEvent struct {
	ID string 		`json:"id"`
	Status string 	`json:"status"`
	Error string	`json:"error,omitempty"`
	Progress string	`json:"progress,omitempty"`
	ProgressDetail struct{
		Current int	`json:"current"`
		Total int	`json:"total"`
	}	`json:"progressDetail"`
}

func DisplayResp(resp io.Reader) error {
	var event *pushEvent
	decoder := json.NewDecoder(resp)

	cursor := Cursor{}
	layers := make([]string, 0)
	prevIndex := 0
	cursor.hide()

	for {
		if err := decoder.Decode(&event); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		// Check if the line is one of the final one
		if strings.Contains(event.Status, "digest") {
			fmt.Printf("%s\n", event.Status)
			break
		}

		// Calculate which line to output
		imageID := event.ID
		currIndex := -1
		for i, id := range layers {
			if id == imageID {
				currIndex = i
				break
			}
		}
		if currIndex == -1 {
			currIndex = len(layers)
			layers = append(layers, event.ID)
		}

		// Move the cursor
		diff := currIndex - prevIndex
		if diff > 0 {
			cursor.moveDown(diff)
		} else if diff < 0 {
			cursor.moveUp(-1 * diff)
		}

		// Output info at the currIndex-th line
		cursor.clearLine()
		if event.Status == "Pushed" {
			fmt.Printf("%s: %s\n", event.ID, event.Status)
		} else if event.ID == "" {
			fmt.Printf("%s %s\n", event.Status, event.Progress)
		} else {
			fmt.Printf("%s: %s %s\n", event.ID, event.Status, event.Progress)
		}

		// because of \n
		prevIndex = currIndex + 1
	}
	return nil
}
