package logger_test

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	applogger "my-bbs/internal/logger"

	"github.com/gin-gonic/gin"
)

func TestLoggerPublicLifecycle(t *testing.T) {
	previousLogWriter := log.Writer()
	previousLogFlags := log.Flags()
	previousLogPrefix := log.Prefix()
	previousGinWriter := gin.DefaultWriter
	previousGinErrorWriter := gin.DefaultErrorWriter

	dir := t.TempDir()
	if err := applogger.Init(dir); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	t.Cleanup(func() {
		applogger.Close()
		log.SetOutput(previousLogWriter)
		log.SetFlags(previousLogFlags)
		log.SetPrefix(previousLogPrefix)
		gin.DefaultWriter = previousGinWriter
		gin.DefaultErrorWriter = previousGinErrorWriter
	})

	const marker = "public-lifecycle-marker"
	applogger.Info(marker)

	const (
		workers = 8
		writes  = 32
	)
	start := make(chan struct{})
	var waitGroup sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		waitGroup.Add(1)
		go func(worker int) {
			defer waitGroup.Done()
			<-start
			for write := 0; write < writes; write++ {
				applogger.Info("worker=%d write=%d", worker, write)
			}
		}(worker)
	}

	waitGroup.Add(1)
	go func() {
		defer waitGroup.Done()
		<-start
		applogger.Close()
	}()

	close(start)
	waitGroup.Wait()
	applogger.Close()

	logFiles, err := filepath.Glob(filepath.Join(dir, "*.log"))
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	if len(logFiles) == 0 {
		t.Fatal("Close() did not leave a log file")
	}

	var contents strings.Builder
	for _, filename := range logFiles {
		data, err := os.ReadFile(filename)
		if err != nil {
			t.Fatalf("ReadFile(%q) error = %v", filename, err)
		}
		contents.Write(data)
	}
	if !strings.Contains(contents.String(), marker) {
		t.Fatalf("Close() did not flush the log written through Info(): %s", contents.String())
	}
}
