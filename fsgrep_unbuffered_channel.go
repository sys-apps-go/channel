package main

import (
	"flag"
	"fmt"
	"github.com/sys-apps-go/channel/common"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

type fsGrep struct {
	wg               sync.WaitGroup
	mu               sync.Mutex
	paths            chan string
}

func main() {
	n := flag.Int("n", runtime.NumCPU(), "Number of worker goroutines")
	k := flag.Int("k", runtime.NumCPU(), "Number of kernel threads running worker goroutines")

	if len(os.Args[1:]) < 2 {
		fmt.Println("Usage: fsgrep_unbuffered_channel <pattern> <dir> [ -n <goroutines> -k <kernel threads>]")
		os.Exit(1)
	}

	pattern := os.Args[1]
	dir := os.Args[2]

	flag.CommandLine.Parse(os.Args[3:])
	numGoroutines := *n
	numKernelThreads := *k

	f := &fsGrep{
		paths:            make(chan string),
	}

	runtime.GOMAXPROCS(numKernelThreads)

	f.wg.Add(numGoroutines + 1)
	for i := 0; i < numGoroutines; i++ {
		go f.searchPatternInFile(pattern)
	}
	go f.getFilePaths(dir)
	f.wg.Wait()
}

func (f *fsGrep) getFilePaths(root string) {
	defer f.wg.Done()
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		f.paths <- path
		return nil
	})
	close(f.paths)
}

func (f *fsGrep) searchPatternInFile(pattern string) {
	defer f.wg.Done()
	for path := range f.paths {
		common.SearchInFile(path, pattern)
	}
}
