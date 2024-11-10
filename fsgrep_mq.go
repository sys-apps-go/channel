package main

import (
	"flag"
	"fmt"
	"github.com/sys-apps-go/channel/common"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
)

type message interface{}

type messageQueue struct {
	messages []message
	mutex    sync.Mutex
	cond     *sync.Cond
	closed   bool
}

type fsGrep struct {
	wg               sync.WaitGroup
	mu               sync.Mutex
	paths            chan []string
	done             atomic.Bool
	queue            *messageQueue
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
		paths:            make(chan []string),
	}

	runtime.GOMAXPROCS(numKernelThreads)

	f.queue = NewMessageQueue()

	f.wg.Add(numGoroutines + 1)
	for i := 0; i < numGoroutines; i++ {
		go f.searchPatternInFile(pattern)
	}
	go f.getFilePaths(dir, chArraySize)
	f.wg.Wait()
}

func (f *fsGrep) getFilePaths(root string, chArraySize int) {
	defer f.wg.Done()

	chArray := make([]string, 0)
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		chArray = append(chArray, path)
		if len(chArray) >= chArraySize {
			f.queue.Send(chArray)
			chArray = make([]string, 0)
		}
		return nil
	})
	if len(chArray) > 0 {
		f.queue.Send(chArray)
	}
	f.queue.Close()
}

func (f *fsGrep) searchPatternInFile(pattern string) {
	defer f.wg.Done()
	for {
		msgs, ok := f.queue.Receive()
		if !ok {
			return
		}
		paths, ok := msgs.([]string)
		if !ok {
			continue // or return, depending on your logic
		}
		for _, path := range paths {
			common.SearchInFile(path, pattern)
		}
	}
}

func NewMessageQueue() *messageQueue {
	mq := &messageQueue{}
	mq.cond = sync.NewCond(&mq.mutex)
	return mq
}

func (mq *messageQueue) Send(msg message) {
	mq.mutex.Lock()
	defer mq.mutex.Unlock()

	if mq.closed {
		return
	}

	mq.messages = append(mq.messages, msg)
	mq.cond.Signal()
}

func (mq *messageQueue) Receive() (message, bool) {
	mq.mutex.Lock()
	defer mq.mutex.Unlock()

	for len(mq.messages) == 0 && !mq.closed {
		mq.cond.Wait()
	}

	if len(mq.messages) == 0 {
		return nil, false
	}

	msg := mq.messages[0]
	mq.messages = mq.messages[1:]
	return msg, true
}

func (mq *messageQueue) Close() {
	mq.mutex.Lock()
	defer mq.mutex.Unlock()

	mq.closed = true
	mq.cond.Broadcast()
}
