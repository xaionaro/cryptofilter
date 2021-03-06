package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/xaionaro-go/secureio"
)

type pipeReadWriter struct {
	Prefix string
	io.Reader
	io.Writer
}

func (p *pipeReadWriter) Close() error {
	return nil
}

func (p *pipeReadWriter) Read(b []byte) (int, error) {
	n, err := p.Reader.Read(b)
	fmt.Println(p.Prefix+" R", n, err, b)
	return n, err
}

func (p *pipeReadWriter) Write(b []byte) (int, error) {
	n, err := p.Writer.Write(b)
	fmt.Println(p.Prefix+" W", n, err, b)
	return n, err
}

type testLogger struct {
	string
}

func (l *testLogger) Error(sess *secureio.Session, err error) bool {
	fmt.Print(err)
	return false
}
func (l *testLogger) Infof(fm string, args ...interface{}) {
	fmt.Printf(l.string+" [I] "+fm+"\n", args...)
}
func (l *testLogger) Debugf(fm string, args ...interface{}) {
	fmt.Printf(l.string+" [D] "+fm+"\n", args...)
}
func (l *testLogger) OnConnect(*secureio.Session) {
}
func (l *testLogger) IsDebugEnabled() bool {
	return true
}

func fatalError(err error) {
	if err == nil {
		return
	}
	panic(err)
}

func main() {
	dir := `/tmp/.test_xaionaro-go_secureio_session_`
	_ = os.Mkdir(dir+"0", 0700)
	_ = os.Mkdir(dir+"1", 0700)
	identity0, xerr := secureio.NewIdentity(dir + "0")
	fatalError(xerr)

	identity1, xerr := secureio.NewIdentity(dir + "1")
	fatalError(xerr)

	pipeR0, pipeW0, err := os.Pipe()
	fatalError(err)

	pipeR1, pipeW1, err := os.Pipe()
	fatalError(err)

	conn0 := &pipeReadWriter{
		Prefix: "0",
		Reader: pipeR0,
		Writer: pipeW1,
	}

	conn1 := &pipeReadWriter{
		Prefix: "1",
		Reader: pipeR1,
		Writer: pipeW0,
	}

	ctx := context.Background()

	sess0 := identity0.NewSession(identity1, conn0, &testLogger{"0"}, nil)
	fatalError(sess0.Start(ctx))

	sess1 := identity1.NewSession(identity0, conn1, &testLogger{"1"}, nil)
	fatalError(sess1.Start(ctx))

	fmt.Println("write")

	_, err = sess0.Write([]byte(`test`))
	fatalError(err)

	fmt.Println("wrote")

	r := make([]byte, 4)
	_, err = sess1.Read(r)
	fatalError(err)

	if string(r) != `test` {
		fatalError(fmt.Errorf(`received string not equals to "test"`))
	}
}
