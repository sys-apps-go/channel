package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"flag"
	"github.com/sys-apps-go/channel/common"
)

type fsGrep struct {
	wg            sync.WaitGroup
	mu            sync.Mutex
	paths         chan []string
	numGoroutines int
	numKernelThreads int
}

func main() {
	n := flag.Int("n", runtime.NumCPU(), "Number of worker goroutines")
	k := flag.Int("k", runtime.NumCPU(), "Number of kernel threads running worker goroutines")
	c := flag.Int("c", 8, "Channel array size")

	if len(os.Args[1:]) < 2 {
		fmt.Println("Usage: fsgrep_unbuffered_channel <pattern> <dir> [ -n <goroutines> -k <kernel threads> -c <channel array size>]")
		os.Exit(1)
	}

	pattern := os.Args[1]
	dir := os.Args[2]

	flag.CommandLine.Parse(os.Args[3:])
	numGoroutines := *n
	numKernelThreads := *k
	chArraySize := *c

	f := &fsGrep{
		paths: make(chan []string, 1024),
		numGoroutines: numGoroutines,
		numKernelThreads: numKernelThreads,
	}

	runtime.GOMAXPROCS(numKernelThreads)

	f.wg.Add(numGoroutines + 1)
	for i := 0; i < numGoroutines; i++ {
		go f.searchPatternInFile(pattern)
	}
	go f.getFilePaths(dir, chArraySize)
	f.wg.Wait()
}

func (f *fsGrep) getFilePaths(root string, chArraySize int) {
	defer f.wg.Done()

	chArray := make([]string,  0)
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		chArray = append(chArray, path)
		if len(chArray) >= chArraySize {
			f.paths <- chArray[0:len(chArray)]
			chArray = make([]string, 0)
		}
		return nil
	})
	if len(chArray) > 0 {
		f.paths <- chArray[0:len(chArray)]
	}
	close(f.paths)
}

func (f *fsGrep) searchPatternInFile(pattern string) {
	defer f.wg.Done()
	for paths := range f.paths {
		for _, path := range paths {
			common.SearchInFile(path, pattern)
		}
	}
}
