package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
	"github.com/gin-gonic/gin"
)

const defaultQueueSize = 1024

var (
	logFile *os.File
	writer  *asyncWriter
)

// asyncWriter 基于 channel 的异步日志写入器（生产者-消费者模型）
// 业务协程只负责把日志投递进 channel，后台 goroutine 负责真正写磁盘/控制台
type asyncWriter struct {
	ch     chan []byte
	out    io.Writer
	wg     sync.WaitGroup
	closed bool
	mu     sync.Mutex
}

func newAsyncWriter(out io.Writer, queueSize int) *asyncWriter {
	w := &asyncWriter{
		ch:  make(chan []byte, queueSize),
		out: out,
	}
	w.wg.Add(1)
	go w.consume()
	return w
}

// Write 实现 io.Writer：把日志内容拷贝后投递到 channel（不阻塞业务太久）
// 注意：必须 copy，因为标准库 log 会复用底层 buffer
func (w *asyncWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	closed := w.closed
	w.mu.Unlock()
	if closed {
		return w.out.Write(p)
	}

	buf := make([]byte, len(p))
	copy(buf, p)

	select {
	case w.ch <- buf:
		// 投递成功，由消费协程异步落盘
	default:
		// channel 满了：降级为同步写，避免丢日志，同时给请求侧一点背压
		_, _ = fmt.Fprint(os.Stderr, "[logger] queue full, fallback to sync write\n")
		_, err := w.out.Write(buf)
		return len(p), err
	}
	return len(p), nil
}

// consume 消费协程：串行从 channel 取日志并写入目标
func (w *asyncWriter) consume() {
	defer w.wg.Done()
	for msg := range w.ch {
		_, _ = w.out.Write(msg)
	}
}

// close 关闭 channel，等待消费协程把剩余日志写完
func (w *asyncWriter) close() {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return
	}
	w.closed = true
	w.mu.Unlock()

	close(w.ch)
	w.wg.Wait()
}

// Init 初始化异步日志：写入当前目录下的日志文件夹，同时输出到控制台
func Init(dir string) error {
	if dir == "" {
		dir = "logs"
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("创建日志目录失败: %w", err)
	}

	filename := filepath.Join(dir, time.Now().Format("2006-01-02")+".log")
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %w", err)
	}
	logFile = f

	mw := io.MultiWriter(os.Stdout, f)
	writer = newAsyncWriter(mw, defaultQueueSize)

	log.SetOutput(writer)
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	// Gin 默认日志也走同一套异步写入
	gin.DefaultWriter = writer
	gin.DefaultErrorWriter = writer

	log.Printf("日志初始化完成（异步 channel），输出目录: %s, queue=%d", dir, defaultQueueSize)
	return nil
}

// Info / Warn / Error 封装，方便业务侧调用（底层仍走异步 channel）
func Info(format string, v ...any) {
	log.Printf("[INFO] "+format, v...)
}

func Warn(format string, v ...any) {
	log.Printf("[WARN] "+format, v...)
}

func Error(format string, v ...any) {
	log.Printf("[ERROR] "+format, v...)
}

// Close 关闭异步日志：排空 channel 后关闭文件（进程退出前必须调用）
func Close() {
	if writer != nil {
		writer.close()
	}
	if logFile != nil {
		_ = logFile.Close()
	}
}
