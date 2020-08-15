package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"flag"
	"io"
	golog "log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"reflect"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/pkg/errors"
)

var (
	secret []byte
	path   string
	dir    string

	poolTimeout int

	gitPullLock sync.Mutex
	gitPullErr  bytes.Buffer

	log    *golog.Logger
	logErr *golog.Logger
)

func main() {
	log = golog.New(os.Stdout, "", golog.LstdFlags)
	logErr = golog.New(os.Stderr, "", golog.LstdFlags)

	argSecret := flag.String("secret", "", "WebHook Secret")
	flag.StringVar(&dir, "dir", "", "Directory of repository")
	flag.StringVar(&path, "path", "/push", "WebHook path")

	argUnix := flag.String("unix", "", "Bind to UNIX Socket")
	argUnixPerm := flag.Int("unix-perm", 777, "UNIX Socket permissions")
	argBind := flag.String("bind", "localhost:8081", "Bind to tcp")

	flag.IntVar(&poolTimeout, "timeout", 30000, "")

	flag.Parse()

	secret = []byte(*argSecret)

	var err error
	if dir == "" {
		log.Println("dir is empty")
		flag.Usage()
		os.Exit(1)
	}
	if _, err = os.Open(dir); err != nil {
		log.Println(err)
		flag.Usage()
		os.Exit(1)
	}

	var l net.Listener

	if *argUnix != "" {
		os.Remove(*argUnix)

		l, err = net.Listen("unix", *argUnix)
		if err != nil {
			log.Fatal(errors.WithStack(err))
		}
		defer l.Close()

		log.Println("Listen unix", *argUnix)

		defer os.Remove(*argUnix)

		perm := os.FileMode((*argUnixPerm / 100 * 16) + (*argUnixPerm / 10 % 10 * 8) + (*argUnixPerm % 10))
		err = os.Chmod(*argUnix, perm)
		if err != nil {
			log.Fatal(errors.WithStack(err))
		}
	} else {
		l, err = net.Listen("tcp", *argBind)
		if err != nil {
			log.Fatal(errors.WithStack(err))
		}
		defer l.Close()

		log.Println("Listen tcp", *argBind)
	}

	server := http.Server{
		ErrorLog: logErr,
		Handler:  http.HandlerFunc(handler),
	}

	go func() {
		err := server.Serve(l)
		if err != nil && err != http.ErrServerClosed {
			log.Fatal(errors.WithStack(err))
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	<-sig

	server.Shutdown(context.TODO())
}

func s2b(s string) (b []byte) {
	sh := *(*reflect.StringHeader)(unsafe.Pointer(&s))
	bh := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	bh.Data = sh.Data
	bh.Len = sh.Len
	bh.Cap = sh.Len
	return b
}
func b2s(b []byte) (s string) {
	return *(*string)(unsafe.Pointer(&b))
}

func handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || !strings.HasPrefix(r.RequestURI, path) {
		log.Println(r.Method, r.RequestURI, "Bad Request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	XHubSignature := r.Header.Get("x-hub-signature")
	if len(XHubSignature) != 45 || !strings.HasPrefix(XHubSignature, "sha1=") {
		log.Println("invalid x-hub-signature")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	h := hmac.New(sha1.New, secret)
	_, err := io.Copy(h, r.Body)
	if err != nil && err != io.EOF {
		logErr.Println(err)
		return
	}

	// sha1=
	var buf [40]byte
	hex.Encode(buf[:], h.Sum(nil))

	if !bytes.EqualFold(buf[:], s2b(XHubSignature)[5:]) {
		log.Println("Invalid signature")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	go func() {
		gitPullLock.Lock()
		defer gitPullLock.Unlock()

		cmd := exec.Cmd{
			Dir:  dir,
			Path: "/bin/bash",
			Args: []string{
				"/bin/bash", "-c", "/usr/bin/git pull -q",
			},
		}

		cmdErr, err := cmd.StderrPipe()
		if cmdErr != nil {
			defer cmdErr.Close()
		}

		err = cmd.Start()
		if err != nil {
			logErr.Println(errors.WithStack(err))
			return
		}

		log.Println("git-pull started")

		chCmdErrDone := make(chan struct{}, 1)
		if cmdErr != nil {
			go func() {
				_, _ = io.Copy(&gitPullErr, cmdErr)
				chCmdErrDone <- struct{}{}
			}()
		} else {
			chCmdErrDone <- struct{}{}
		}

		ch := make(chan error, 1)
		go func() {
			ch <- cmd.Wait()
		}()

		select {
		case <-time.After(time.Duration(poolTimeout) * time.Millisecond):
			printGitErr(chCmdErrDone)

			log.Println("git-pull timeout.")
			err = cmd.Process.Kill()
			if err != nil {
				logErr.Println(err)
			}

		case err = <-ch:
			printGitErr(chCmdErrDone)

			log.Println("git-pull exited.")
			if gitPullErr.Len() > 0 {
				logErr.Println(gitPullErr.Bytes())
			}

			if err != nil {
				logErr.Println(err)
			}
		}
	}()
}

func printGitErr(chCmdErrDone <-chan struct{}) {
	<-chCmdErrDone
	defer gitPullErr.Reset()

	firstLine := true

	br := bufio.NewReader(&gitPullErr)
	for {
		line, _, err := br.ReadLine()
		if err != nil {
			break
		}

		if firstLine {
			logErr.Println(("--------------------------------------------------"))
			logErr.Println("git-pull stderr:")
			defer logErr.Println("--------------------------------------------------")
			firstLine = false
		}

		logErr.Println(b2s(line))
	}
}
