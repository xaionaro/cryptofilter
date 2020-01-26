package secureio

import (
	"fmt"
	"io"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xaionaro-go/errors"
)

const (
	testUseUnixSocket = true
)

type testLogger struct {
	string
	*testing.T
	enableInfo  bool
	enableDebug bool
	Session     *Session
}

func (l *testLogger) Error(sess *Session, err error) bool {
	xerr := err.(*errors.Error)
	if xerr.Has(io.EOF) {
		return false
	}
	l.T.Errorf("E:%v:SID:%v:%v", l.T.Name(), sess.ID(), xerr)
	return false
}
func (l *testLogger) Infof(format string, args ...interface{}) {
	if !l.enableInfo {
		fmt.Printf("I:%v:SID:%v: "+format+"\n",
			append([]interface{}{l.T.Name(), l.Session.ID()}, args...)...)
		return
	}
	l.T.Errorf(l.string+" [I] "+format, args...)
}
func (l *testLogger) Debugf(format string, args ...interface{}) {
	fmt.Printf("D:%v:SID:%v: "+format+"\n",
		append([]interface{}{l.T.Name(), l.Session.ID()}, args...)...)
}
func (l *testLogger) OnConnect(sess *Session) {
}
func (l *testLogger) OnInit(sess *Session) {
	l.Session = sess
}
func (l *testLogger) IsDebugEnabled() bool {
	return l.enableDebug
}

type pipeReadWriter struct {
	Prefix string
	io.ReadCloser
	io.WriteCloser
}

func (p *pipeReadWriter) Close() error {
	p.ReadCloser.Close()
	p.WriteCloser.Close()
	return nil
}

func (p *pipeReadWriter) Read(b []byte) (int, error) {
	n, err := p.ReadCloser.Read(b)
	return n, err
}

func (p *pipeReadWriter) Write(b []byte) (int, error) {
	n, err := p.WriteCloser.Write(b)

	return n, err
}

var testPairMutex sync.Mutex

func testPair(t *testing.T) (identity0, identity1 *Identity, conn0, conn1 io.ReadWriteCloser) {
	var err error

	go func() {
		http.ListenAndServe("localhost:6060", nil)
	}()

	dir := `/tmp/.test_xaionaro-go_secureio_session_`
	_ = os.Mkdir(dir+"0", 0700)
	_ = os.Mkdir(dir+"1", 0700)
	identity0, err = NewIdentity(dir + "0")
	if err != nil {
		t.Fatal(err)
	}
	identity1, err = NewIdentity(dir + "1")
	if err != nil {
		t.Fatal(err)
	}

	if testUseUnixSocket {

		testPairMutex.Lock()
		defer testPairMutex.Unlock()

		os.Remove(`/tmp/xaionaro-go-secureio-0.sock`)

		l0, err := net.Listen(`unixpacket`, `/tmp/xaionaro-go-secureio-0.sock`)
		if err != nil {
			t.Fatal(err)
		}

		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			defer wg.Done()
			conn1, err = net.Dial(`unixpacket`, `/tmp/xaionaro-go-secureio-0.sock`)
			if err != nil {
				t.Fatal(err)
			}
		}()

		conn0, err = l0.Accept()
		if err != nil {
			t.Fatal(err)
		}

		wg.Wait()

	} else {

		pipeR0, pipeW0, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}

		pipeR1, pipeW1, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}

		conn0 = &pipeReadWriter{
			Prefix:      "0",
			ReadCloser:  pipeR0,
			WriteCloser: pipeW1,
		}

		conn1 = &pipeReadWriter{
			Prefix:      "1",
			ReadCloser:  pipeR1,
			WriteCloser: pipeW0,
		}

	}

	return
}

func testConnIsOpen(t *testing.T, conn0, conn1 io.ReadWriteCloser) {
	b := []byte(`test`)
	_, err := conn0.Write(b)
	assert.NoError(t, err)

	_, err = conn1.Write(b)
	assert.NoError(t, err)

	_, err = conn0.Read(b)
	assert.NoError(t, err)
	assert.Equal(t, `test`, string(b))

	_, err = conn1.Read(b)
	assert.NoError(t, err)
	assert.Equal(t, `test`, string(b))
}
