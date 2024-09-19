package memory

import (
	"fmt"
	"time"

	"github.com/shirou/gopsutil/mem"
)

const memoryThreshold float64 = 0.1 

func MonitorMemoryUsage(workers []chan int) {
	for {
		v, _ := mem.VirtualMemory()

		freeMemoryPercent := float64(v.Available) / float64(v.Total)

		if freeMemoryPercent < memoryThreshold {

			setState(workers, 0)
			fmt.Printf("Free memory below threshold (%.2f%%). Pausing workers...\n", freeMemoryPercent*100)

			for freeMemoryPercent < memoryThreshold {
				fmt.Println(freeMemoryPercent)
				time.Sleep(1 * time.Second)
				v, _ = mem.VirtualMemory()
				freeMemoryPercent = float64(v.Available) / float64(v.Total)
			}
			fmt.Println("Free memory above threshold. Resuming workers.")
			setState(workers, 1)
		}

		time.Sleep(500 * time.Millisecond)
	}
}

func setState(workers []chan int, state int) {
	for _, w := range workers {
		w <- state
	}
}

