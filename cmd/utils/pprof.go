package utils

import (
	"fmt"
	"os"
	"runtime/pprof"
	"time"
)

var (
	cpuFile = NewCpuFile()
)

type CpuFile struct {
	isStart   bool
	isEnd     bool
	startTime time.Time
	file      *os.File
}

func NewCpuFile() *CpuFile {
	file, err := os.OpenFile("cpu.prof", os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		panic(err)
	}
	return &CpuFile{file: file}
}

func (c *CpuFile) handle(number uint64) {
	if c.isStart {
		if !c.isEnd && number >= 0 {
			pprof.StopCPUProfile()
			if err := c.file.Close(); err != nil {
				panic(err)
			}
			c.isEnd = true
			fmt.Println("StopCpuFile", number, "ts", time.Now().Sub(c.startTime).Seconds())
		}
		return
	}

	if number >= 500 {
		c.isStart = true
		if err := pprof.StartCPUProfile(c.file); err != nil {
			panic(err)
		}
		c.startTime = time.Now()
		fmt.Println("StartCpuProfile", number)
	}
}
