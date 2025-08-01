package network

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"database/sql/driver"
	"encoding/binary"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sijms/go-ora/v2/configurations"

	"github.com/sijms/go-ora/v2/trace"

	"github.com/sijms/go-ora/v2/converters"
)

// var ErrConnectionReset error = errors.New("connection reset")
var read_buffer_size = 0x4000

type Data interface {
	Write(session *Session) error
	Read(session *Session) error
}

type SessionState struct {
	summary   *SummaryObject
	sendPcks  []PacketInterface
	InBuffer  []byte
	OutBuffer bytes.Buffer
	index     int
}

type Session struct {
	mu             sync.Mutex
	conn           net.Conn
	sslConn        *tls.Conn
	reader         *bufio.Reader
	remainingBytes int
	lastPacket     bytes.Buffer
	Context        *SessionContext
	sendPcks       []PacketInterface
	inBuffer       []byte
	outBuffer      bytes.Buffer
	index          int
	breakIndex     int
	// doneContext       []chan struct{}
	TimeZone          []byte
	TTCVersion        uint8
	HasEOSCapability  bool
	HasFSAPCapability bool
	Summary           *SummaryObject
	states            []SessionState
	StrConv           converters.IStringConverter
	UseBigClrChunks   bool
	UseBigScn         bool
	ClrChunkSize      int
	breakConn         bool
	Connected         bool
	// resetConn         bool
	SSL struct {
		CertificateRequest []*x509.CertificateRequest
		PrivateKeys        []*rsa.PrivateKey
		Certificates       []*x509.Certificate
		roots              *x509.CertPool
		tlsCertificates    []tls.Certificate
	}
	tracer trace.Tracer
}

func NewSessionWithInputBufferForDebug(input []byte) *Session {
	options := &configurations.ConnectionConfig{
		AdvNegoServiceInfo: configurations.AdvNegoServiceInfo{AuthService: nil},
		SessionInfo: configurations.SessionInfo{
			SessionDataUnitSize:   0xFFFF,
			TransportDataUnitSize: 0xFFFF,
		},
	}
	return &Session{
		// ctx:        context.Background(),
		conn:       nil,
		inBuffer:   input,
		index:      0,
		lastPacket: bytes.Buffer{},
		// connOption:      *connOption,
		Context:         NewSessionContext(options),
		Summary:         nil,
		UseBigClrChunks: false,
		ClrChunkSize:    0x40,
		tracer:          trace.NilTracer(),
	}
}

func NewSession(config *configurations.ConnectionConfig, tracer trace.Tracer) *Session {
	return &Session{
		// ctx:             context.Background(),
		conn:            nil,
		inBuffer:        nil,
		index:           0,
		Context:         NewSessionContext(config),
		Summary:         nil,
		UseBigClrChunks: false,
		ClrChunkSize:    0x40,
		lastPacket:      bytes.Buffer{},
		tracer:          tracer,
	}
}

// SaveState save current session state and accept new state
// if new state is nil the session will be reset
func (session *Session) SaveState(newState *SessionState) {
	session.mu.Lock()
	defer session.mu.Unlock()
	session.states = append(session.states, SessionState{
		summary:   session.Summary,
		sendPcks:  session.sendPcks,
		InBuffer:  session.inBuffer,
		OutBuffer: session.outBuffer,
		index:     session.index,
	})
	if newState == nil {
		session.Summary = nil
		session.sendPcks = nil
		session.inBuffer = nil
		session.outBuffer = bytes.Buffer{}
		session.index = 0
	} else {
		session.Summary = newState.summary
		session.sendPcks = newState.sendPcks
		session.inBuffer = newState.InBuffer
		session.outBuffer = newState.OutBuffer
		session.index = newState.index
	}
}

// LoadState load last saved session state and return the current state
// if this is the only session state available set session state memory to nil
func (session *Session) LoadState() (oldState *SessionState) {
	index := len(session.states) - 1
	if index >= 0 {
		oldState = &session.states[index]
	}
	if index >= 0 {
		currentState := session.states[index]
		session.Summary = currentState.summary
		session.sendPcks = currentState.sendPcks
		session.inBuffer = currentState.InBuffer
		session.outBuffer = currentState.OutBuffer
		session.index = currentState.index
		if index == 0 {
			session.states = nil
		} else {
			session.states = session.states[:index]
		}
	}
	return
}

func (session *Session) resetWrite() {
	session.mu.Lock()
	session.sendPcks = nil
	session.outBuffer.Reset()
	session.mu.Unlock()
}

func (session *Session) resetRead() {
	session.inBuffer = nil
	session.index = 0
}

// ResetBuffer empty in and out buffer and set read index to 0
func (session *Session) ResetBuffer() {
	session.Summary = nil
	session.resetWrite()
	session.resetRead()
}

func (session *Session) SetConnected() {
	session.mu.Lock()
	defer session.mu.Unlock()
	session.Connected = true
}

func (session *Session) StartContext(ctx context.Context) chan struct{} {
	//session.oldCtx = session.ctx
	//session.ctx = ctx
	done := make(chan struct{})
	//session.doneContext = append(session.doneContext, done)
	go func(idone chan struct{}, mu *sync.Mutex) {
		var err error
		mu.Lock()
		tracer := session.tracer
		mu.Unlock()
		select {
		case <-idone:
			return
		case <-ctx.Done():
			select {
			case <-done:
				return
			default:
				session.mu.Lock()
				connected := session.Connected
				session.mu.Unlock()
				if connected {
					if err = session.BreakConnection(); err != nil {
						tracer.Print("Connection Break Error: ", err)
					}
				} else {
					err = session.WriteFinalPacket()
					if err != nil {
						tracer.Print("Write Final Packet With Error: ", err)
					}
					session.Disconnect()
				}
			}
		}
	}(done, &session.mu)
	return done
}

func (session *Session) EndContext(done chan struct{}) {
	if done != nil {
		close(done)
	}
	// index := len(session.doneContext) - 1
	// if index >= 0 {
	// 	done := session.doneContext[index]
	// 	if done != nil {
	// 		close(done)
	// 		done = nil
	// 	}
	// }
	// if index == 0 {
	// 	session.doneContext = nil
	// } else {
	// 	session.doneContext = session.doneContext[:index]
	// }
	// if session.doneContext != nil {
	// 	close(session.doneContext)
	// 	session.doneContext = nil
	// }
	// session.ctx = session.oldCtx
}

func (session *Session) initRead() error {
	var err error
	timeout := time.Time{}
	if session.Context.connConfig.Timeout > 0 {
		timeout = time.Now().Add(session.Context.connConfig.Timeout)
	}
	// if deadline, ok := session.ctx.Deadline(); ok && !session.IsBreak() {
	// 	timeout = deadline
	// }
	if session.sslConn != nil {
		err = session.sslConn.SetReadDeadline(timeout)
	} else if session.conn != nil {
		err = session.conn.SetReadDeadline(timeout)
	} else {
		return errors.New("attempt to set timeout on closed connection")
	}
	return err
}

func (session *Session) initWrite() error {
	var err error
	timeout := time.Time{}
	if session.Context.connConfig.Timeout > 0 {
		timeout = time.Now().Add(session.Context.connConfig.Timeout)
	}
	// if deadline, ok := session.ctx.Deadline(); ok && !session.IsBreak() {
	// 	timeout = deadline
	// }
	if session.sslConn != nil {
		err = session.sslConn.SetWriteDeadline(timeout)
	} else if session.conn != nil {
		err = session.conn.SetWriteDeadline(timeout)
	} else {
		return errors.New("attempt to set timeout on closed connection")
	}
	return err
}

// LoadSSLData load data required for SSL connection like certificate, private keys and
// certificate requests
func (session *Session) LoadSSLData(certs, keys, certRequests [][]byte) error {
	for _, temp := range certs {
		cert, err := x509.ParseCertificate(temp)
		if err != nil {
			return err
		}
		session.SSL.Certificates = append(session.SSL.Certificates, cert)
		for _, temp2 := range keys {
			key, err := x509.ParsePKCS1PrivateKey(temp2)
			if err != nil {
				return err
			}
			if key.PublicKey.Equal(cert.PublicKey) {
				certPem := pem.EncodeToMemory(&pem.Block{
					Type:  "CERTIFICATE",
					Bytes: temp,
				})
				keyPem := pem.EncodeToMemory(&pem.Block{
					Type:  "RSA PRIVATE KEY",
					Bytes: x509.MarshalPKCS1PrivateKey(key),
				})
				tlsCert, err := tls.X509KeyPair(certPem, keyPem)
				if err != nil {
					return err
				}
				session.SSL.tlsCertificates = append(session.SSL.tlsCertificates, tlsCert)
			}
		}
	}
	for _, temp := range certRequests {
		cert, err := x509.ParseCertificateRequest(temp)
		if err != nil {
			return err
		}
		session.SSL.CertificateRequest = append(session.SSL.CertificateRequest, cert)
	}
	return nil
}

// negotiate it is a step in SSL communication in which tcp connection is
// used to create sslConn object
func (session *Session) negotiate() {
	connOption := session.Context.connConfig
	host := connOption.GetActiveServer(false)

	if tlsConfig := connOption.TLSConfig; tlsConfig != nil {
		tlsConfig.ServerName = host.Addr
		session.sslConn = tls.Client(session.conn, tlsConfig)
		return
	}

	if session.SSL.roots == nil && len(session.SSL.Certificates) > 0 {
		session.SSL.roots = x509.NewCertPool()
		for _, cert := range session.SSL.Certificates {
			session.SSL.roots.AddCert(cert)
		}
	}
	config := &tls.Config{
		ServerName: host.Addr,
	}
	if len(session.SSL.tlsCertificates) > 0 {
		config.Certificates = session.SSL.tlsCertificates
	}
	if session.SSL.roots != nil {
		config.RootCAs = session.SSL.roots
	}
	if !connOption.SSLVerify {
		config.InsecureSkipVerify = true
	}
	session.sslConn = tls.Client(session.conn, config)
}

func (session *Session) ResetBreak() {
	session.mu.Lock()
	session.breakConn = false
	session.mu.Unlock()
}

// IsBreak tell if the connection break elicit
func (session *Session) IsBreak() bool {
	return session.breakConn
}

// func (session *Session) resetConnection() (PacketInterface, error) {
// 	temp, err := session.readPacket()
// 	if err != nil {
// 		return nil, err
// 	}
// 	if pck, ok := temp.(*MarkerPacket); ok {
// 		switch pck.markerType {
// 		case 0:
// 			session.breakConn = true
// 		case 1:
// 			if pck.markerData == 2 {
// 				session.resetConn = true
// 			} else {
// 				session.breakConn = true
// 			}
// 		default:
// 			return nil, errors.New("unknown marker type")
// 		}
// 	} else {
// 		return nil, errors.New("marker packet not received")
// 	}
// 	err = session.writePacket(newMarkerPacket(2, session.Context))
// 	if err != nil {
// 		return nil, err
// 	}
// 	for session.breakConn && !session.resetConn {
// 		temp, err = session.readPacket()
// 		if pck, ok := temp.(*MarkerPacket); ok {
// 			switch pck.markerType {
// 			case 0:
// 				session.breakConn = true
// 			case 1:
// 				if pck.markerData == 2 {
// 					session.resetConn = true
// 				} else {
// 					session.breakConn = true
// 				}
// 			default:
// 				return nil, errors.New("unknown marker type")
// 			}
// 		} else {
// 			return nil, errors.New("marker packet not received")
// 		}
// 	}
// 	session.ResetBuffer()
// 	if session.resetConn && session.Context.AdvancedService.HashAlgo != nil {
// 		err = session.Context.AdvancedService.HashAlgo.Init()
// 		if err != nil {
// 			return nil, err
// 		}
// 	}
// 	if session.resetConn && session.Context.AdvancedService.CryptAlgo != nil {
// 		err = session.Context.AdvancedService.CryptAlgo.Reset()
// 		if err != nil {
// 			return nil, err
// 		}
// 	}
// 	session.breakConn = false
// 	session.resetConn = false
// 	return session.readPacket()
// }

func (session *Session) RestoreIndex() bool {
	if session.index < session.breakIndex {
		session.index = session.breakIndex
		return true
	}
	return false
}

// BreakConnection elicit connection break to cancel the current operation
func (session *Session) BreakConnection() error {
	session.tracer.Print("Break Connection")
	var err error
	// first discard remaining bytes
	// if session.remainingBytes > 0 {
	// 	_, err = session.readPacket()
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// }

	done := false
	if session.Context.NegotiatedOptions&0x400 > 0 {
		done, err = sendOOB(session.conn)
		if err != nil {
			return err
		}
	}
	if !done {
		err = session.writePacket(newMarkerPacket(marker_type_interrupt, session.Context))
		if err != nil {
			return err
		}
	}
	session.mu.Lock()
	session.breakConn = true
	session.mu.Unlock()
	return nil
	// return session.readPacket()
	// session.ResetBuffer()
	// if done {
	// 	err = session.writePacket(newMarkerPacket(2, session.Context))
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// } else {
	//
	// }
	// ret := make([]PacketInterface, 0, 2)
	// var pck PacketInterface
	// pck, err = session.readPacket()
	// if err != nil {
	// 	return nil, err
	// }
	// ret = append(ret, pck)
	// for pck != nil && pck.getFlag()&0x20 > 0 {
	// 	pck, err = session.readPacket()
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	ret = append(ret, pck)
	// }
}

// Connect perform network connection on address:port
// check if the client need to use SSL
// then send connect packet to the server and
// receive either accept, redirect or refuse packet
func (session *Session) Connect(ctx context.Context) error {
	connOption := session.Context.connConfig
	session.ResetBuffer()
	session.Disconnect()
	session.tracer.Print("Connect")
	var err error
	connected := false
	var host *configurations.ServerAddr
	loop := true
	dialer := connOption.Dialer
	if dialer == nil {
		dialer = &net.Dialer{}
		if session.Context.connConfig.ConnectTimeout > 0 {
			dialer = &net.Dialer{
				Timeout: session.Context.connConfig.ConnectTimeout,
			}
		} else {
			dialer = &net.Dialer{}
		}
	}
	// connOption.serverIndex = 0
	for loop {
		host = connOption.GetActiveServer(false)
		if host == nil {
			if err != nil {
				return err
			}
			return errors.New("no available servers to connect to")
		}
		addr := host.NetworkAddr()
		if len(session.Context.connConfig.UnixAddress) > 0 {
			session.conn, err = dialer.DialContext(ctx, "unix", session.Context.connConfig.UnixAddress)
		} else {
			session.conn, err = dialer.DialContext(ctx, "tcp", addr)
		}

		if err != nil {
			session.tracer.Printf("using: %s ..... [FAILED]", addr)
			host = connOption.GetActiveServer(true)
			if host == nil {
				break
			}
			continue
		}
		session.tracer.Printf("using: %s ..... [SUCCEED]", addr)
		connected = true
		loop = false
	}
	if !connected {
		return err
	}
	err = connOption.UpdateSSL(host)
	if err != nil {
		return err
	}
	if connOption.SSL {
		session.tracer.Print("Using SSL/TLS")
		session.negotiate()
		session.reader = bufio.NewReaderSize(session.sslConn, read_buffer_size)
	} else {
		session.reader = bufio.NewReaderSize(session.conn, read_buffer_size)
	}
	session.tracer.Print("Open :", connOption.ConnectionData())
	connectPacket := newConnectPacket(session.Context)
	err = session.writePacket(connectPacket)
	if err != nil {
		return err
	}
	if uint16(connectPacket.length) == connectPacket.dataOffset {
		session.PutBytes(connectPacket.buffer...)
		err = session.Write()
		if err != nil {
			return err
		}
	}
	pck, err := session.readPacket()
	if err != nil {
		return err
	}

	if acceptPacket, ok := pck.(*AcceptPacket); ok {
		session.mu.Lock()
		*session.Context = *acceptPacket.sessionCtx
		session.Context.handshakeComplete = true
		session.mu.Unlock()
		session.tracer.Print("Handshake Complete")
		return nil
	}
	if redirectPacket, ok := pck.(*RedirectPacket); ok {
		session.tracer.Print("Redirect")
		err = session.Context.connConfig.UpdateDatabaseInfoForRedirect(redirectPacket.redirectAddr, redirectPacket.reconnectData)
		if err != nil {
			return err
		}
		session.Context.connConfig.ResetServerIndex()
		session.Context.isRedirect = true
		return session.Connect(ctx)
	}
	if refusePacket, ok := pck.(*RefusePacket); ok {
		refusePacket.extractErrCode()
		var addr string
		var port int
		if host != nil {
			addr = host.Addr
			port = host.Port
		}
		session.tracer.Printf("connection to %s:%d refused with error: %s", addr, port, refusePacket.Err.Error())
		host = connOption.GetActiveServer(true)
		if host == nil {
			session.Disconnect()
			return refusePacket.Err
		}
		return session.Connect(ctx)
	}
	return errors.New("connection refused by the server due to unknown reason")
}

func (session *Session) WriteFinalPacket() error {
	data, err := newDataPacket(nil, session.Context, session.tracer, &session.mu)
	if err != nil {
		return err
	}
	data.dataFlag = 0x40
	return session.writePacket(data)
}

// Disconnect close the network and release resources
func (session *Session) Disconnect() {
	session.mu.Lock()
	defer session.mu.Unlock()
	// session.ResetBuffer()
	if session.sslConn != nil {
		_ = session.sslConn.Close()
		session.sslConn = nil
	}
	if session.conn != nil {
		_ = session.conn.Close()
		session.conn = nil
	}
}

// Write send data store in output buffer through network
//
// if data bigger than fSessionDataUnit it should be divided into
// segment and each segment sent in data packet
func (session *Session) Write() error {
	outputBytes := session.outBuffer.Bytes()
	size := session.outBuffer.Len()
	if size == 0 {
		// send empty data packet
		pck, err := newDataPacket(nil, session.Context, session.tracer, &session.mu)
		if err != nil {
			return err
		}
		return session.writePacket(pck)
		// return errors.New("the output buffer is empty")
	}

	segmentLen := int(session.Context.SessionDataUnit - 64)
	offset := 0
	if size > segmentLen {
		segment := make([]byte, segmentLen)
		for size > segmentLen {
			copy(segment, outputBytes[offset:offset+segmentLen])
			pck, err := newDataPacket(segment, session.Context, session.tracer, &session.mu)
			if err != nil {
				return err
			}
			err = session.writePacket(pck)
			if err != nil {
				session.outBuffer.Reset()
				return err
			}
			size -= segmentLen
			offset += segmentLen
		}
	}
	if size != 0 {
		pck, err := newDataPacket(outputBytes[offset:], session.Context, session.tracer, &session.mu)
		if err != nil {
			return err
		}
		err = session.writePacket(pck)
		if err != nil {
			session.outBuffer.Reset()
			return err
		}
	}
	return nil
}

func (session *Session) processMarker() error {
	var err error
	// send reset connection
	session.resetWrite()
	marker := newMarkerPacket(marker_type_reset, session.Context)
	err = session.writePacket(marker)
	if err != nil {
		return err
	}
	if session.Context.AdvancedService.HashAlgo != nil {
		err = session.Context.AdvancedService.HashAlgo.Init()
		if err != nil {
			return err
		}
	}
	if session.Context.AdvancedService.CryptAlgo != nil {
		err = session.Context.AdvancedService.CryptAlgo.Reset()
		if err != nil {
			return err
		}
	}
	_, err = session.readPacket()
	if err != nil {
		return err
	}
	// receive all packet until you get marker
	// var pck PacketInterface = nil
	// for pck == nil || pck.getPacketType() != MARKER {
	//
	// 	if session.isFinalPacketRead {
	// 		break
	// 	}
	// }

	// receive all marker packet
	// for pck != nil && pck.getPacketType() == MARKER && !session.isFinalPacketRead {
	// 	pck, err = session.readPacket()
	// 	if err != nil {
	// 		return err
	// 	}
	// }

	return nil
	// breakConn, resetConn := false, false

	// switch input.markerType {
	// case 0:
	// 	breakConn = true
	// case marker_type_break:
	// 	if input.markerData == 2 {
	// 		resetConn = true
	// 	} else {
	// 		breakConn = true
	// 	}
	// default:
	// 	return fmt.Errorf("unknown marker type: %d", input.markerType)
	// }
	// trials := 1
	// for breakConn && !resetConn {
	// 	if trials > 5 { return errors.New("connection break")}
	// 	pck, err := session.readPacket()
	// 	if err != nil {
	// 		return err
	// 	}
	// 	if mPck, ok := pck.(*MarkerPacket); ok {
	// 		switch mPck.markerType {
	// 		case 0:
	// 			breakConn = true
	// 		case marker_type_break:
	// 			if mPck.markerData == 2 {
	// 				resetConn = true
	// 			} else {
	// 				breakConn = true
	// 			}
	// 		default:
	// 			return fmt.Errorf("unknown marker type: %d", mPck.markerType)
	// 		}
	// 	}else {
	// 		return errors.New("connection break")
	// 	}
	// 	trials ++
	// }
}

func (session *Session) Peek(numBytes int) ([]byte, error) {
	oldIndex := session.index
	ret, err := session.read(numBytes)
	session.index = oldIndex
	return ret, err
}

// Read numBytes of data from input buffer if requested data is larger
// than input buffer session will get the remaining from network stream
func (session *Session) read(numBytes int) ([]byte, error) {
	var err error
	var pck PacketInterface

	requiredLen := numBytes

	// first discard remaining bytes
	// if session.remainingBytes > 0 {
	// 	_, err = session.readPacket()
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// }
	for session.index+numBytes > len(session.inBuffer) {
		// if break connection send break message
		if session.IsBreak() {
			// _, err = session.BreakConnection()
			// if err != nil {
			// 	return nil, err
			// }
			session.ResetBreak()
		}
		pck, err = session.readPacket()
		if err != nil {
			// if e, ok := err.(net.Error); ok && e.Timeout() {
			// 	var breakErr error
			// 	tracer.Print("Read Timeout")
			// 	pck, breakErr = session.BreakConnection()
			// 	if breakErr != nil {
			// 		// return nil, err
			// 		tracer.Print("Connection Break With Error: ", breakErr)
			// 		return nil, err
			// 	}
			// } else {
			// 	return nil, err
			// }
			return nil, err
		}

		// if session.IsBreak() {
		// 	for pck == nil {
		// 		pck, err = session.readPacket()
		// 		if err != nil {
		// 			return nil, err
		// 		}
		// 	}
		// } else {
		// 	if pck == nil {
		// 		continue
		// 	}
		// }
		if pck == nil {
			continue
		}
		if _, ok := pck.(*MarkerPacket); ok {
			session.breakIndex = len(session.inBuffer)
			err = session.processMarker()
			if err != nil {
				return nil, err
			}
			session.index = session.breakIndex
			// session.ResetBreak()
			return nil, ErrConnReset
		} else {
			return nil, fmt.Errorf("receive abnormal packet type %d instead of data packet", pck.getPacketType())
		}
		// pcks := []PacketInterface{pck}
		// for _, pck := range pcks {
		// 	if dataPck, ok := pck.(*DataPacket); ok {
		// 		session.inBuffer = append(session.inBuffer, dataPck.buffer...)
		// 	} else {
		// 		return nil, errors.New("the packet received is not data packet")
		// 	}
		// }
		// if session.IsBreak() {
		// 	if session.index+numBytes > len(session.inBuffer) {
		// 		numBytes = len(session.inBuffer) - session.index
		// 	}
		// 	break
		// }
	}
	// for session.index+numBytes > len(session.inBuffer) {
	// 	tempPck, err := session.readPacket()
	// 	if err != nil {
	// 		if e, ok := err.(net.Error); ok && e.Timeout() {
	// 			session.Context.connConfig.Tracer.Print("Read Timeout")
	// 			var breakErr error
	// 			tempPck, breakErr = session.BreakConnection()
	// 			if breakErr != nil {
	// 				// return nil, err
	// 				session.Context.connConfig.Tracer.Print("Connection Break With Error: ", breakErr)
	// 				return nil, err
	// 			}
	// 		} else {
	// 			return nil, err
	// 		}
	// 	}
	// 	loop := true
	// 	for loop {
	// 		switch pck := tempPck.(type) {
	// 		case *DataPacket:
	// 			session.inBuffer = append(session.inBuffer, pck.buffer...)
	// 			loop = false
	// 		case *MarkerPacket:
	// 			tempPck, err = session.resetConnection()
	// 		default:
	// 			return nil, errors.New("the packet received is not data packet")
	// 		}
	// 	}
	// }
	ret := session.inBuffer[session.index : session.index+numBytes]
	session.index += numBytes
	if len(ret) < requiredLen {
		return nil, errors.New("buffer under-run during read operation")
	}
	return ret, nil
}

// Write a packet to the network stream
func (session *Session) writePacket(pck PacketInterface) error {
	session.mu.Lock()
	defer session.mu.Unlock()
	session.sendPcks = append(session.sendPcks, pck)
	tmp := pck.bytes()
	session.tracer.LogPacket("Write packet:", tmp)
	err := session.initWrite()
	if err != nil {
		return err
	}
	if session.sslConn != nil {
		_, err = session.sslConn.Write(tmp)
	} else if session.conn != nil {
		_, err = session.conn.Write(tmp)
	} else {
		return errors.New("attempt to write on closed connection")
	}
	return err
}

// HasError Check if the session has error or not
func (session *Session) HasError() bool {
	return session.Summary != nil && (session.Summary.RetCode != 0 && session.Summary.RetCode != 1403)
}

// GetError Return the error in form or OracleError
func (session *Session) GetError() *OracleError {
	err := &OracleError{}
	if session.HasError() {
		err.ErrCode = session.Summary.RetCode
		if session.StrConv != nil {
			err.ErrMsg = session.StrConv.Decode(session.Summary.ErrorMessage)
		} else {
			err.ErrMsg = string(session.Summary.ErrorMessage)
		}
		err.errPos = session.Summary.errorPos
	}
	return err
}

func (session *Session) readAll(size int) error {
	// session.mu.Lock()
	// defer session.mu.Unlock()
	index := 0
	tempBuffer := make([]byte, size)
	var err error
	var temp int
	session.remainingBytes = size
	for index < size {
		if session.conn == nil && session.sslConn == nil {
			return errors.New("closed connection")
		}
		err = session.initRead()
		if err != nil {
			return err
		}
		temp, err = session.reader.Read(tempBuffer[index:])
		if err != nil {
			if temp > 0 {
				session.remainingBytes -= temp
				index += temp
			}
			session.lastPacket.Write(tempBuffer[:index])
			return err
		}
		session.remainingBytes -= temp
		index += temp
	}
	session.lastPacket.Write(tempBuffer)
	return nil
}

// readPacketData
func (session *Session) readPacketData() error {
	var err error
	if session.remainingBytes > 0 {
		if session.lastPacket.Len() < 8 { // means break occur inside head
		}
		err = session.readAll(session.remainingBytes)
		if err != nil {
			return err
		}
		session.remainingBytes = 0
		return nil
	}
	session.lastPacket.Reset()
	err = session.readAll(8)
	if err != nil {
		// if remaining bytes = 8 means no data read so make it 0
		if session.remainingBytes == 8 {
			session.remainingBytes = 0
		}
		return err
	}
	head := session.lastPacket.Bytes()
	var length uint32
	// pckType := PacketType(head[4])
	// flag := head[5]

	if session.Context.handshakeComplete && session.Context.Version >= 315 {
		length = binary.BigEndian.Uint32(head)
	} else {
		length = uint32(binary.BigEndian.Uint16(head))
	}
	length -= 8
	err = session.readAll(int(length))
	if err != nil {
		return err
	}
	session.tracer.LogPacket("Read packet:", session.lastPacket.Bytes())
	return nil
}

// read a packet from network stream
func (session *Session) readPacket() (PacketInterface, error) {
	var err error
	err = session.readPacketData()
	if err != nil {
		return nil, err
	}
	packetData := session.lastPacket.Bytes()
	pckType := PacketType(packetData[4])
	flag := packetData[5]
	// session.isFinalPacketRead = flag&0x20 > 0
	// log.Printf("Response: %#v\n\n", packetData)
	switch pckType {
	case RESEND:
		if session.Context.connConfig.SSL && flag&8 != 0 {
			session.negotiate()
			session.reader = bufio.NewReaderSize(session.sslConn, read_buffer_size)
		}
		resend := func() error {
			session.mu.Lock()
			defer session.mu.Unlock()
			for _, pck := range session.sendPcks {
				err := session.initWrite()
				if err != nil {
					return err
				}
				if session.sslConn != nil {
					_, err = session.sslConn.Write(pck.bytes())
				} else if session.conn != nil {
					_, err = session.conn.Write(pck.bytes())
				} else {
					err = errors.New("attempt to write on closed connection")
				}
				if err != nil {
					return err
				}
			}
			return nil
		}
		// for _, pck := range session.sendPcks {
		// 	//log.Printf("Request: %#v\n\n", pck.bytes())
		// 	err := session.initWrite()
		// 	if err != nil {
		// 		return nil, err
		// 	}
		// 	if session.sslConn != nil {
		// 		_, err = session.sslConn.Write(pck.bytes())
		// 	} else if session.conn != nil {
		// 		_, err = session.conn.Write(pck.bytes())
		// 	} else {
		// 		return nil, errors.New("attempt to write on closed connection")
		// 	}
		// 	if err != nil {
		// 		return nil, err
		// 	}
		// }
		err = resend()
		if err != nil {
			return nil, err
		}
		return session.readPacket()
	case ACCEPT:
		return newAcceptPacketFromData(packetData, session.Context.connConfig), nil
	case REFUSE:
		return newRefusePacketFromData(packetData), nil
	case REDIRECT:
		pck := newRedirectPacketFromData(packetData)
		dataLen := binary.BigEndian.Uint16(packetData[8:])
		var data string
		if uint16(pck.length) <= pck.dataOffset {
			err = session.readPacketData()
			packetData = session.lastPacket.Bytes()
			dataPck, err := newDataPacketFromData(packetData, session.Context, session.tracer, &session.mu)
			if err != nil {
				return nil, err
			}
			data = string(dataPck.buffer)
		} else {
			data = string(packetData[10 : 10+dataLen])
		}
		// fmt.Println("data returned: ", data)
		length := strings.Index(data, "\x00")
		if pck.flag&2 != 0 && length > 0 {
			pck.redirectAddr = data[:length]
			pck.reconnectData = data[length:]
		} else {
			pck.redirectAddr = data
		}
		return pck, nil
	case DATA:
		dataPck, err := newDataPacketFromData(packetData, session.Context, session.tracer, &session.mu)
		if dataPck != nil {
			if session.Context.connConfig.SSL && (dataPck.dataFlag&0x8000 > 0 || dataPck.flag&0x80 > 0) {
				session.negotiate()
				session.reader = bufio.NewReaderSize(session.sslConn, read_buffer_size)
			}
			if dataPck.dataFlag == 0x40 {
				// close connection
				session.Disconnect()
				err = driver.ErrBadConn
			}
			session.inBuffer = append(session.inBuffer, dataPck.buffer...)
		}
		return nil, err
	case MARKER:
		return newMarkerPacketFromData(packetData, session.Context), nil
		// switch pck.markerType {
		// case 0:
		// 	session.breakConn = true
		// case 1:
		// 	if pck.markerData == 2 {
		// 		session.resetConn = true
		// 	} else {
		// 		session.breakConn = true
		// 	}
		// default:
		// 	return nil, errors.New("unknown marker type")
		// }
		// trials := 1
		// for session.breakConn && !session.resetConn {
		// 	if trials > 5 {
		// 		return nil, errors.New("connection break")
		// 	}
		// 	packetData, err = readPacketData()
		// 	if err != nil {
		// 		return nil, err
		// 	}
		// 	pck = newMarkerPacketFromData(packetData, session.Context)
		// 	if pck == nil {
		// 		return nil, errors.New("connection break")
		// 	}
		// 	switch pck.markerType {
		// 	case 0:
		// 		session.breakConn = true
		// 	case 1:
		// 		if pck.markerData == 2 {
		// 			session.resetConn = true
		// 		} else {
		// 			session.breakConn = true
		// 		}
		// 	default:
		// 		return nil, errors.New("unknown marker type")
		// 	}
		// 	trials++
		// }
		// session.ResetBuffer()
		// err = session.writePacket(newMarkerPacket(2, session.Context))
		// if err != nil {
		// 	return nil, err
		// }
		// if session.resetConn && session.Context.AdvancedService.HashAlgo != nil {
		// 	err = session.Context.AdvancedService.HashAlgo.Init()
		// 	if err != nil {
		// 		return nil, err
		// 	}
		// }
		// if session.resetConn && session.Context.AdvancedService.CryptAlgo != nil {
		// 	err = session.Context.AdvancedService.CryptAlgo.Reset()
		// 	if err != nil {
		// 		return nil, err
		// 	}
		// }
		// session.breakConn = false
		// session.resetConn = false
		// // return nil, ErrConnectionReset
		// packetData, err = readPacketData()
		// if err != nil {
		//	return nil, err
		// }
		// dataPck, err := newDataPacketFromData(packetData, session.Context)
		// //session.breakConn = true
		// return dataPck, err
		// return newMarkerPacketFromData(packetData, session.Context), nil
		// switch pck.markerType {
		// case 0:
		// 	session.breakConn = true
		// case 1:
		// 	if pck.markerData == 2 {
		// 		session.resetConn = true
		// 	} else {
		// 		session.breakConn = true
		// 	}
		// default:
		// 	return nil, errors.New("unknown marker type")
		// }
		// trials := 1
		// for session.breakConn && !session.resetConn {
		// 	// if trials > 3 {
		// 	// 	return nil, errors.New("connection break")
		// 	// }
		// 	packetData, err = readPacketData()
		// 	if err != nil {
		// 		return nil, err
		// 	}
		// 	pck = newMarkerPacketFromData(packetData, session.Context)
		// 	if pck == nil {
		// 		return nil, errors.New("connection break")
		// 	}
		// 	switch pck.markerType {
		// 	case 0:
		// 		session.breakConn = true
		// 	case 1:
		// 		if pck.markerData == 2 {
		// 			session.resetConn = true
		// 		} else {
		// 			session.breakConn = true
		// 		}
		// 	default:
		// 		return nil, errors.New("unknown marker type")
		// 	}
		// 	trials++
		// }
		// session.ResetBuffer()
		// err = session.writePacket(newMarkerPacket(2, session.Context))
		// if err != nil {
		// 	return nil, err
		// }
		// if session.resetConn && session.Context.AdvancedService.HashAlgo != nil {
		// 	err = session.Context.AdvancedService.HashAlgo.Init()
		// 	if err != nil {
		// 		return nil, err
		// 	}
		// }
		// if session.resetConn && session.Context.AdvancedService.CryptAlgo != nil {
		// 	err = session.Context.AdvancedService.CryptAlgo.Reset()
		// 	if err != nil {
		// 		return nil, err
		// 	}
		// }
		// session.breakConn = false
		// session.resetConn = false
		// //return nil, ErrConnectionReset
		// packetData, err = readPacketData()
		// if err != nil {
		// 	return nil, err
		// }
		// dataPck, err := newDataPacketFromData(packetData, session.Context)
		// return dataPck, err
		// if err != nil {
		// 	return nil, err
		// }
		// if dataPck == nil {
		// 	return nil, errors.New("connection break")
		// }
		//
		// session.inBuffer = dataPck.buffer
		// session.index = 0
		// loop := true
		// for loop {
		// 	msg, err := session.GetByte()
		// 	if err != nil {
		// 		return nil, err
		// 	}
		// 	switch msg {
		// 	case 4:
		// 		loop = false
		// 		session.Summary, err = NewSummary(session)
		// 		if err != nil {
		// 			return nil, err
		// 		}
		// 		if session.HasError() {
		// 			return nil, session.GetError()
		// 		}
		// 	case 8:
		// 		size, err := session.GetInt(2, true, true)
		// 		if err != nil {
		// 			return nil, err
		// 		}
		// 		for x := 0; x < 2; x++ {
		// 			_, err = session.GetInt(4, true, true)
		// 			if err != nil {
		// 				return nil, err
		// 			}
		// 		}
		// 		for x := 2; x < size; x++ {
		// 			_, err = session.GetInt(4, true, true)
		// 			if err != nil {
		// 				return nil, err
		// 			}
		// 		}
		// 		_, err = session.GetInt(2, true, true)
		// 		if err != nil {
		//			return nil, err
		// 		}
		// 		size, err = session.GetInt(2, true, true)
		// 		for x := 0; x < size; x++ {
		// 			_, val, num, err := session.GetKeyVal()
		// 			if err != nil {
		// 				return nil, err
		// 			}
		// 			if num == 163 {
		// 				session.TimeZone = val
		// 			}
		// 		}
		// 		if session.TTCVersion >= 4 {
		// 			// get queryID
		// 			size, err = session.GetInt(4, true, true)
		// 			if err != nil {
		// 				return nil, err
		// 			}
		// 			if size > 0 {
		// 				bty, err := session.GetBytes(size)
		// 				if err != nil {
		// 					return nil, err
		// 				}
		// 				if len(bty) >= 8 {
		// 					queryID := binary.LittleEndian.Uint64(bty[size-8:])
		// 					fmt.Println("query ID: ", queryID)
		// 				}
		// 			}
		// 		}
		// 		if session.TTCVersion >= 7 {
		// 			length, err := session.GetInt(4, true, true)
		// 			if err != nil {
		// 				return nil, err
		// 			}
		// 			for i := 0; i < length; i++ {
		// 				_, err = session.GetInt(8, true, true)
		// 				if err != nil {
		// 					return nil, err
		// 				}
		// 			}
		// 		}
		// 	default:
		// 		return nil, errors.New(fmt.Sprintf("TTC error: received code %d during stmt reading", msg))
		// 	}
		//
		// }
		// fallthrough
	default:
		// fmt.Printf("Packet Data: %#v\n", packetData)
		return nil, fmt.Errorf("unsupported packet type: %d", pckType)
	}
}

// PutString write a string data to output buffer
func (session *Session) PutString(data string) {
	session.PutClr([]byte(data))
}

// GetString read a string data from input buffer
func (session *Session) GetString(length int) (string, error) {
	ret, err := session.GetClr()
	return string(ret[:length]), err
}

// PutBytes write bytes of data to output buffer
func (session *Session) PutBytes(data ...byte) {
	session.outBuffer.Write(data)
}

// PutUint write uint number with size entered either use bigEndian or not and use compression or not to
func (session *Session) PutUint(number interface{}, size uint8, bigEndian, compress bool) {
	var num uint64
	switch number := number.(type) {
	case int64:
		num = uint64(number)
	case int32:
		num = uint64(number)
	case int16:
		num = uint64(number)
	case int8:
		num = uint64(number)
	case uint64:
		num = number
	case uint32:
		num = uint64(number)
	case uint16:
		num = uint64(number)
	case uint8:
		num = uint64(number)
	case uint:
		num = uint64(number)
	case int:
		num = uint64(number)
	default:
		panic("you need to pass an integer to this function")
	}
	// if the size is one byte no compression occur only one byte written
	if size == 1 {
		session.outBuffer.WriteByte(uint8(num))
		// session.OutBuffer = append(session.OutBuffer, uint8(num))
		return
	}
	if compress {
		temp := make([]byte, 8)
		binary.BigEndian.PutUint64(temp, num)
		temp = bytes.TrimLeft(temp, "\x00")
		if size > uint8(len(temp)) {
			size = uint8(len(temp))
		}
		if size == 0 {
			session.outBuffer.WriteByte(0)
			// session.OutBuffer = append(session.OutBuffer, 0)
		} else {
			session.outBuffer.WriteByte(size)
			session.outBuffer.Write(temp)
			// session.OutBuffer = append(session.OutBuffer, size)
			// session.OutBuffer = append(session.OutBuffer, temp...)
		}
	} else {
		temp := make([]byte, size)
		if bigEndian {
			switch size {
			case 2:
				binary.BigEndian.PutUint16(temp, uint16(num))
			case 4:
				binary.BigEndian.PutUint32(temp, uint32(num))
			case 8:
				binary.BigEndian.PutUint64(temp, num)
			}
		} else {
			switch size {
			case 2:
				binary.LittleEndian.PutUint16(temp, uint16(num))
			case 4:
				binary.LittleEndian.PutUint32(temp, uint32(num))
			case 8:
				binary.LittleEndian.PutUint64(temp, num)
			}
		}
		session.outBuffer.Write(temp)
		// session.OutBuffer = append(session.OutBuffer, temp...)
	}
}

// PutInt write int number with size entered either use bigEndian or not and use compression or not to
func (session *Session) PutInt(number interface{}, size uint8, bigEndian bool, compress bool) {
	var num int64
	switch number := number.(type) {
	case int64:
		num = number
	case int32:
		num = int64(number)
	case int16:
		num = int64(number)
	case int8:
		num = int64(number)
	case uint64:
		num = int64(number)
	case uint32:
		num = int64(number)
	case uint16:
		num = int64(number)
	case uint8:
		num = int64(number)
	case uint:
		num = int64(number)
	case int:
		num = int64(number)
	default:
		panic("you need to pass an integer to this function")
	}

	if compress {
		temp := make([]byte, 8)
		binary.BigEndian.PutUint64(temp, uint64(num))
		temp = bytes.TrimLeft(temp, "\x00")
		if size > uint8(len(temp)) {
			size = uint8(len(temp))
		}
		if size == 0 {
			session.outBuffer.WriteByte(0)
			// session.OutBuffer = append(session.OutBuffer, 0)
		} else {
			if num < 0 {
				num = num * -1
				size = size & 0x80
			}
			session.outBuffer.WriteByte(size)
			session.outBuffer.Write(temp)
		}
	} else {
		if size == 1 {
			session.outBuffer.WriteByte(uint8(num))
		} else {
			temp := make([]byte, size)
			if bigEndian {
				switch size {
				case 2:
					binary.BigEndian.PutUint16(temp, uint16(num))
				case 4:
					binary.BigEndian.PutUint32(temp, uint32(num))
				case 8:
					binary.BigEndian.PutUint64(temp, uint64(num))
				}
			} else {
				switch size {
				case 2:
					binary.LittleEndian.PutUint16(temp, uint16(num))
				case 4:
					binary.LittleEndian.PutUint32(temp, uint32(num))
				case 8:
					binary.LittleEndian.PutUint64(temp, uint64(num))
				}
			}
			session.outBuffer.Write(temp)
		}
	}
}

// PutClr write variable length bytearray to output buffer
func (session *Session) PutClr(data []byte) {
	dataLen := len(data)
	if dataLen > 0xFC {
		session.outBuffer.WriteByte(0xFE)
		start := 0
		for start < dataLen {
			end := start + session.ClrChunkSize
			if end > dataLen {
				end = dataLen
			}
			temp := data[start:end]
			if session.UseBigClrChunks {
				session.PutInt(len(temp), 4, true, true)
			} else {
				session.outBuffer.WriteByte(uint8(len(temp)))
			}
			session.outBuffer.Write(temp)
			start += session.ClrChunkSize
		}
		session.outBuffer.WriteByte(0)
	} else if dataLen == 0 {
		session.outBuffer.WriteByte(0)
	} else {
		session.outBuffer.WriteByte(uint8(len(data)))
		session.outBuffer.Write(data)
	}
}

// PutKeyValString write key, val (in form of string) and flag number to output buffer
func (session *Session) PutKeyValString(key string, val string, num uint8) {
	session.PutKeyVal([]byte(key), []byte(val), num)
}

// PutKeyVal write key, val (in form of bytearray) and flag number to output buffer
func (session *Session) PutKeyVal(key []byte, val []byte, num uint8) {
	if len(key) == 0 {
		session.outBuffer.WriteByte(0)
	} else {
		session.PutUint(len(key), 4, true, true)
		session.PutClr(key)
	}
	if len(val) == 0 {
		session.outBuffer.WriteByte(0)
	} else {
		session.PutUint(len(val), 4, true, true)
		session.PutClr(val)
	}
	session.PutInt(num, 4, true, true)
}

// GetByte read one uint8 from input buffer
func (session *Session) GetByte() (uint8, error) {
	rb, err := session.read(1)
	if err != nil {
		return 0, err
	}
	return rb[0], nil
}

// GetInt64 read int64 number from input buffer.
//
// you should specify the size of the int and either compress or not and stored as big endian or not
func (session *Session) GetInt64(size int, compress bool, bigEndian bool) (int64, error) {
	var ret int64
	negFlag := false
	if compress {
		rb, err := session.read(1)
		if err != nil {
			return 0, err
		}
		size = int(rb[0])
		if size&0x80 > 0 {
			negFlag = true
			size = size & 0x7F
		}
		bigEndian = true
	}
	if size == 0 {
		return 0, nil
	} else if size > 8 {
		// When compress is true, "size" may be a value greater than 8 in some cases
		return 0, errors.New("invalid size for GetInt64: " + strconv.Itoa(size))
	}
	rb, err := session.read(size)
	if err != nil {
		return 0, err
	}
	temp := make([]byte, 8)
	if bigEndian {
		copy(temp[8-size:], rb)
		ret = int64(binary.BigEndian.Uint64(temp))
	} else {
		copy(temp[:size], rb)
		// temp = append(pck.buffer[pck.index: pck.index + size], temp...)
		ret = int64(binary.LittleEndian.Uint64(temp))
	}
	if negFlag {
		ret = ret * -1
	}
	return ret, nil
}

// GetInt read int number from input buffer.
// you should specify the size of the int and either compress or not and stored as big endian or not
func (session *Session) GetInt(size int, compress bool, bigEndian bool) (int, error) {
	temp, err := session.GetInt64(size, compress, bigEndian)
	if err != nil {
		return 0, err
	}
	return int(temp), nil
}

// GetNullTermString read a null terminated string from input buffer
func (session *Session) GetNullTermString(maxSize int) (result string, err error) {
	oldIndex := session.index
	temp, err := session.read(maxSize)
	if err != nil {
		return
	}
	find := bytes.Index(temp, []byte{0})
	if find > 0 {
		result = string(temp[:find])
		session.index = oldIndex + find + 1
	} else {
		result = string(temp)
	}
	return
}

// GetClr reed variable length bytearray from input buffer
func (session *Session) GetClr() (output []byte, err error) {
	var nb byte
	nb, err = session.GetByte()
	if err != nil {
		return
	}
	if nb == 0 || nb == 0xFF || nb == 0xFD {
		output = nil
		err = nil
		return
	}
	chunkSize := int(nb)
	var chunk []byte
	var tempBuffer bytes.Buffer
	if chunkSize == 0xFE {
		for chunkSize > 0 {
			//if session.IsBreak() {
			//	break
			//}
			if session.UseBigClrChunks {
				chunkSize, err = session.GetInt(4, true, true)
			} else {
				nb, err = session.GetByte()
				chunkSize = int(nb)
			}
			if err != nil {
				return
			}
			chunk, err = session.GetBytes(chunkSize)
			if err != nil {
				return
			}
			tempBuffer.Write(chunk)
		}
	} else {
		chunk, err = session.GetBytes(chunkSize)
		if err != nil {
			return
		}
		tempBuffer.Write(chunk)
	}
	output = tempBuffer.Bytes()
	return
	// var size uint8
	// var rb []byte
	// size, err = session.GetByte()
	// if err != nil {
	// 	return
	// }
	// if size == 0 || size == 0xFF {
	// 	output = nil
	// 	err = nil
	// 	return
	// }
	// if size != 0xFE {
	// 	output, err = session.read(int(size))
	// 	return
	// }
	//
	// for {
	// 	var size1 int
	// 	if session.UseBigClrChunks {
	// 		size1, err = session.GetInt(4, true, true)
	// 	} else {
	// 		size, err = session.GetByte()
	// 		size1 = int(size)
	// 	}
	// 	if err != nil || size1 == 0 {
	// 		break
	// 	}
	// 	rb, err = session.read(size1)
	// 	if err != nil {
	// 		return
	// 	}
	// 	tempBuffer.Write(rb)
	// }
	// output = tempBuffer.Bytes()
	// return
}

// GetDlc read variable length bytearray from input buffer
func (session *Session) GetDlc() (output []byte, err error) {
	var length int
	length, err = session.GetInt(4, true, true)
	if err != nil {
		return
	}
	if length > 0 {
		output, err = session.GetClr()
		if len(output) > length {
			output = output[:length]
		}
	}
	return
}

// GetBytes read specified number of bytes from input buffer
func (session *Session) GetBytes(length int) ([]byte, error) {
	return session.read(length)
}

// GetKeyVal read key, value (in form of bytearray), a number flag from input buffer
func (session *Session) GetKeyVal() (key []byte, val []byte, num int, err error) {
	key, err = session.GetDlc()
	if err != nil {
		return
	}
	val, err = session.GetDlc()
	if err != nil {
		return
	}
	num, err = session.GetInt(4, true, true)
	return
}

func (session *Session) WriteBytes(buffer *bytes.Buffer, data ...byte) {
	buffer.Write(data)
}

// WriteUint write uint data to external buffer
func (session *Session) WriteUint(buffer *bytes.Buffer, number interface{}, size uint8, bigEndian, compress bool) {
	val := reflect.ValueOf(number)
	var num uint64
	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		num = uint64(val.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		num = val.Uint()
	default:
		panic("you need to pass an integer to this function")
	}
	if size == 1 {
		buffer.WriteByte(uint8(num))
		return
	}
	if compress {
		// if the size is one byte no compression occur only one byte written
		temp := make([]byte, 8)
		binary.BigEndian.PutUint64(temp, num)
		temp = bytes.TrimLeft(temp, "\x00")
		if size > uint8(len(temp)) {
			size = uint8(len(temp))
		}
		if size == 0 {
			buffer.WriteByte(0)
		} else {
			buffer.WriteByte(size)
			buffer.Write(temp)
		}
	} else {
		temp := make([]byte, size)
		if bigEndian {
			switch size {
			case 2:
				binary.BigEndian.PutUint16(temp, uint16(num))
			case 4:
				binary.BigEndian.PutUint32(temp, uint32(num))
			case 8:
				binary.BigEndian.PutUint64(temp, num)
			}
		} else {
			switch size {
			case 2:
				binary.LittleEndian.PutUint16(temp, uint16(num))
			case 4:
				binary.LittleEndian.PutUint32(temp, uint32(num))
			case 8:
				binary.LittleEndian.PutUint64(temp, num)
			}
		}
		buffer.Write(temp)
	}
}

// WriteInt write int data to external buffer
func (session *Session) WriteInt(buffer *bytes.Buffer, number interface{}, size uint8, bigEndian, compress bool) {
	val := reflect.ValueOf(number)
	var num int64
	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		num = val.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		num = int64(val.Uint())
	default:
		panic("you need to pass an integer to this function")
	}
	if compress {
		temp := make([]byte, 8)
		binary.BigEndian.PutUint64(temp, uint64(num))
		temp = bytes.TrimLeft(temp, "\x00")
		if size > uint8(len(temp)) {
			size = uint8(len(temp))
		}
		if size == 0 {
			buffer.WriteByte(0)
		} else {
			if num < 0 {
				num = num * -1
				size = size & 0x80
			}
			buffer.WriteByte(size)
			buffer.Write(temp)
		}
	} else {
		if size == 1 {
			buffer.WriteByte(uint8(num))
		} else {
			temp := make([]byte, size)
			if bigEndian {
				switch size {
				case 2:
					binary.BigEndian.PutUint16(temp, uint16(num))
				case 4:
					binary.BigEndian.PutUint32(temp, uint32(num))
				case 8:
					binary.BigEndian.PutUint64(temp, uint64(num))
				}
			} else {
				switch size {
				case 2:
					binary.LittleEndian.PutUint16(temp, uint16(num))
				case 4:
					binary.LittleEndian.PutUint32(temp, uint32(num))
				case 8:
					binary.LittleEndian.PutUint64(temp, uint64(num))
				}
			}
			buffer.Write(temp)
		}
	}
}

func (session *Session) GetFixedClr() ([]byte, error) {
	nb, err := session.GetByte()
	if err != nil {
		return nil, err
	}
	var size int
	switch nb {
	case 0, 0xFD, 0xFF:
		return nil, nil
	case 0xFE:
		size, err = session.GetInt(4, false, true)
		if err != nil {
			return nil, err
		}
	default:
		size = int(nb)
	}
	return session.GetBytes(size)
}
func (session *Session) WriteFixedClr(buffer *bytes.Buffer, data []byte) {
	if len(data) > 0xFC {
		buffer.WriteByte(0xFE)
		session.WriteUint(buffer, len(data), 4, true, false)
	} else {
		buffer.WriteByte(uint8(len(data)))
	}
	buffer.Write(data)
}

func (session *Session) WriteClr(buffer *bytes.Buffer, data []byte) {
	dataLen := len(data)
	if dataLen > 0xFC {
		buffer.WriteByte(0xFE)
		start := 0
		for start < dataLen {
			end := start + session.ClrChunkSize
			if end > dataLen {
				end = dataLen
			}
			temp := data[start:end]
			if session.UseBigClrChunks {
				session.WriteInt(buffer, len(temp), 4, true, true)
			} else {
				buffer.WriteByte(uint8(len(temp)))
			}
			buffer.Write(temp)
			start += session.ClrChunkSize
		}
		buffer.WriteByte(0)
	} else if dataLen == 0 {
		buffer.WriteByte(0)
	} else {
		buffer.WriteByte(uint8(len(data)))
		buffer.Write(data)
	}
}

// WriteKeyValString write key, val (in form of string) and flag number to external buffer
func (session *Session) WriteKeyValString(buffer *bytes.Buffer, key string, val string, num uint8) {
	session.WriteKeyVal(buffer, []byte(key), []byte(val), num)
}

// WriteKeyVal write key, val, flag number to external buffer
func (session *Session) WriteKeyVal(buffer *bytes.Buffer, key []byte, val []byte, num uint8) {
	if len(key) == 0 {
		buffer.WriteByte(0)
	} else {
		session.WriteUint(buffer, len(key), 4, true, true)
		session.WriteClr(buffer, key)
	}
	if len(val) == 0 {
		buffer.WriteByte(0)
		// session.OutBuffer = append(session.OutBuffer, 0)
	} else {
		session.WriteUint(buffer, len(val), 4, true, true)
		session.WriteClr(buffer, val)
	}
	session.WriteInt(buffer, num, 4, true, true)
}

// func (session *Session) ReadInt64(buffer *bytes.Buffer, size int, compress, bigEndian bool) (int64, error) {
// 	var ret int64
// 	negFlag := false
// 	if compress {
// 		rb, err := buffer.ReadByte()
// 		if err != nil {
// 			return 0, err
// 		}
// 		size = int(rb)
// 		if size&0x80 > 0 {
// 			negFlag = true
// 			size = size & 0x7F
// 		}
// 		bigEndian = true
// 	}
// 	if size == 0 {
// 		return 0, nil
// 	}
// 	tempBytes, err := session.ReadBytes(buffer, size)
// 	if err != nil {
// 		return 0, err
// 	}
// 	temp := make([]byte, 8)
// 	if bigEndian {
// 		copy(temp[8-size:], tempBytes)
// 		ret = int64(binary.BigEndian.Uint64(temp))
// 	} else {
// 		copy(temp[:size], tempBytes)
// 		ret = int64(binary.LittleEndian.Uint64(temp))
// 	}
// 	if negFlag {
// 		ret = ret * -1
// 	}
// 	return ret, nil
// }
//
// func (session *Session) ReadInt(buffer *bytes.Buffer, size int, compress, bigEndian bool) (int, error) {
// 	temp, err := session.ReadInt64(buffer, size, compress, bigEndian)
// 	return int(temp), err
// }
//
// func (session *Session) ReadBytes(buffer *bytes.Buffer, size int) ([]byte, error) {
// 	temp := make([]byte, size)
// 	_, err := buffer.Read(temp)
// 	return temp, err
// }
//
// func (session *Session)ReadClr(buffer *bytes.Buffer) (output []byte, err error){
// 	var size uint8
// 	var rb []byte
// 	size, err = buffer.ReadByte()
// 	if err != nil {
// 		return
// 	}
// 	if size == 0 || size == 0xFF {
// 		output = nil
//   		err = nil
// 		return
// 	}
// 	if size != 0xFE {
// 		output, err = session.ReadBytes(buffer, int(size))//  session.read(int(size))
// 		return
// 	}
// 	var tempBuffer bytes.Buffer
// 	for {
// 		var size1 int
// 		if session.UseBigClrChunks {
// 			size1, err = session.ReadInt(buffer, 4, true, true)
// 		} else {
// 			size1, err = session.ReadInt(buffer, 1, false, false)
// 		}
// 		if err != nil || size1 == 0 {
// 			break
// 		}
// 		rb, err = session.ReadBytes(buffer, size1)
// 		if err != nil {
// 			return
// 		}
// 		tempBuffer.Write(rb)
// 	}
// 	output = tempBuffer.Bytes()
// 	return
// }
//
// func (session *Session)ReadDlc(buffer *bytes.Buffer) (output []byte, err error) {
// 	var length int
// 	length, err = session.ReadInt(buffer, 4, true, true)
// 	if err != nil {
// 		return
// 	}
// 	if length > 0 {
// 		output, err = session.ReadClr(buffer)
// 		if len(output) > length {
// 			output = output[:length]
// 		}
// 	}
// 	return
// }
