package common

import (
	"fmt"
	"runtime"
	"io"
	"os"
	"bufio"
	"path/filepath"
	"strings"
	"bytes"
	"syscall"
	"unsafe"
	"runtime/pprof"
	"github.com/fatih/color"
)

// Set CPU affinity for the process (PID pid) to CPU ...
func setCPUAffinity(pid int, cpus ...int) error {
	var mask uint64
	for _, cpu := range cpus {
		mask |= 1 << cpu
	}

	// Call sched_setaffinity
	_, _, errno := syscall.RawSyscall(
		syscall.SYS_SCHED_SETAFFINITY,
		uintptr(pid),
		uintptr(unsafe.Sizeof(mask)),
		uintptr(unsafe.Pointer(&mask)),
	)

	if errno != 0 {
		return fmt.Errorf("failed to set CPU affinity: %v", errno)
	}

	return nil
}

func SearchInFile(path, pattern string) error {
	var err error
	isText, err := isTextFile(path)
	if !isText || err != nil {
		return err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	start := 0
	for {
		index := bytes.Index(data[start:], []byte(pattern))
		if index < 0 {
			break
		}

		// Calculate the absolute index in the original data
		absIndex := start + index

		// Find the start and end of the line
		lineStart := bytes.LastIndex(data[:absIndex], []byte("\n")) + 1
		lineEnd := bytes.Index(data[absIndex:], []byte("\n"))

		// If it's the last line, set lineEnd to the end of data
		if lineEnd == -1 {
			lineEnd = len(data)
		} else {
			lineEnd += absIndex
		}

		// Print the line with colored pattern
		line := data[lineStart:lineEnd]
		coloredLine := bytes.Replace(line, []byte(pattern), []byte(color.RedString(pattern)), -1)

		// Print line number in blue
		lineNumber := bytes.Count(data[:absIndex], []byte("\n")) + 1
		fmt.Printf("%s(%s) %s\n", path, color.BlueString("%d", lineNumber), coloredLine)

		// Move start forward to continue searching for pattern
		start = lineEnd
	}
	return nil
}

func StartCPUProfile(cpuFile string) {
	// Create files to store the profiles
	cpuProfileFile, err := os.Create(cpuFile)
	if err != nil {
		fmt.Println("Error creating CPU profile file:", err)
		return
	}

	// Start CPU profiling
	err = pprof.StartCPUProfile(cpuProfileFile)
	if err != nil {
		fmt.Println("Error starting CPU profile:", err)
		return
	}
}

func StartMemoryProfile(memFile string) {
	memProfileFile, err := os.Create(memFile)
	if err != nil {
		fmt.Println("Error creating memory profile file:", err)
		return
	}
	// Write memory profile
	runtime.GC() // Run garbage collection to get accurate memory usage
	err = pprof.WriteHeapProfile(memProfileFile)
	if err != nil {
		fmt.Println("Error writing memory profile:", err)
		return
	}

}

func isTextFile(filename string) (bool, error) {
	file, err := os.Open(filename)
	if err != nil {
		return false, err
	}
	defer file.Close()

	reader := bufio.NewReader(file)

	// Read the first 512 bytes
	buf := make([]byte, 512)
	n, err := reader.Read(buf)
	if err != nil && err != io.EOF {
		return false, err
	}

	// Check for null bytes and non-printable characters
	nullCount := 0
	for _, b := range buf[:n] {
		if b == 0 {
			nullCount++
		}
		if b < 0x20 && b != 0x09 && b != 0x0A && b != 0x0D {
			//if (b < 7 || b > 14) && (b < 32 || b > 127) {
			return false, nil // Likely binary
		}
	}

	// If more than 30% null bytes, consider it binary
	if float64(nullCount)/float64(n) > 0.3 {
		return false, nil
	}

	return true, nil // Likely text
}

func IsHidden(path string) bool {
	// Get the base name of the file or directory
	name := filepath.Base(path)

	// Check if it starts with a dot (Unix-style hidden files)
	if strings.HasPrefix(name, ".") {
		return true
	}
	return false
}
