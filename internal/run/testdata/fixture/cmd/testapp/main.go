// Command testapp is a throwaway fixture for internal/run's Ginkgo specs —
// not part of sgo itself. It prints the GREETING env var on a tight loop so
// a test can observe both that it's alive and which environment it started
// (or restarted) with.
package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	greeting := os.Getenv("GREETING")
	for {
		fmt.Println("GREETING=" + greeting)
		time.Sleep(20 * time.Millisecond)
	}
}
