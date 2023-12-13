package lib

import (
	"io"
	"log"
	"os"
	"strings"
	"time"
)

type RotatingFileWriter struct {
	logFilePathValidUntil int64
	logFile               *os.File
	Folder                string
	Pattern               string
}

func (r *RotatingFileWriter) RotateLogFile() {
	if r.logFile != nil {
		_ = r.logFile.Close()
	}
	now := time.Now()
	nowMS := now.UnixMilli()
	tenMinutesMS := int64(10 * 60 * 1000)
	fromMS := (nowMS / tenMinutesMS) * tenMinutesMS
	untilMS := fromMS + tenMinutesMS

	from := time.Unix(fromMS/1000, 0)

	originalUtcIsoDate := from.Format(time.RFC3339)
	filePostfix := originalUtcIsoDate[0:15] + "0"
	filePostfix = strings.Replace(filePostfix, " ", "", -1)
	filePostfix = strings.Replace(filePostfix, ":", "", -1)
	filePostfix = strings.Replace(filePostfix, "T", "", -1)
	filePostfix = strings.Replace(filePostfix, "-", "", -1)

	path := r.Folder
	if !strings.HasSuffix(r.Pattern, "/") {
		path += "/"
	}
	path += strings.Replace(r.Pattern, "{TIME}", filePostfix, -1)
	var err error
	r.logFile, err = os.OpenFile(path, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0600)

	if err != nil {
		panic(err)
	}

	header := os.Getenv("CONVERGENCE_LOG_FILE_HEADER")
	host, err := os.Hostname()
	r.logFile.Write([]byte("------------------------------------------------------------"))
	if header != "" {
		r.logFile.Write([]byte("\nHeader: " + header))
	}
	r.logFile.Write([]byte("\nHost: " + host))

	r.logFile.Write([]byte("\n------------------------------------------------------------\n\n\n"))
	r.logFilePathValidUntil = untilMS
}

func (r *RotatingFileWriter) Write(p []byte) (n int, err error) {
	if time.Now().UnixMilli() > r.logFilePathValidUntil {
		r.RotateLogFile()
	}

	return r.logFile.Write(p)
}

func (r *RotatingFileWriter) Close() error {
	if r.logFile != nil {
		return r.logFile.Close()
	}

	return nil
}

func InitiateLogFileRedirection() func() {
	path := ServiceInstance.GetConfiguration("observability.path").(string)
	pattern := ServiceInstance.GetConfiguration("observability.stdout").(string)

	rotatingFileWriter := &RotatingFileWriter{
		Folder:  path,
		Pattern: pattern,
	}
	rotatingFileWriter.RotateLogFile()

	out := os.Stdout
	mw := io.MultiWriter(out, rotatingFileWriter)

	// get pipe reader and writer | writes to pipe writer come out pipe reader
	r, w, _ := os.Pipe()

	// replace stdout,stderr with pipe writer | all writes to stdout, stderr will go through pipe instead (fmt.print, log)
	os.Stdout = w
	os.Stderr = w

	// writes with log.Print should also write to mw
	log.SetOutput(mw)

	//create channel to control exit | will block until all copies are finished
	exit := make(chan bool)

	go func() {
		// copy all reads from pipe to multiwriter, which writes to stdout and file
		_, _ = io.Copy(mw, r)
		// when r or w is closed copy will finish and true will be sent to channel
		exit <- true
	}()

	// function to be deferred in main until program exits
	return func() {
		// close writer then block on exit channel | this will let mw finish writing before the program exits
		_ = w.Close()
		<-exit
		// close file after all writes have finished
		_ = rotatingFileWriter.Close()
	}
}
