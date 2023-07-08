package server

import (
	"net/http"
	"time"
)

func waitForContainerToStart(url string) {
	for {
		_, err := http.Get(url)
		if err == nil {
			// container is up and running
			return
		}

		time.Sleep(100 * time.Millisecond)
	}
}
