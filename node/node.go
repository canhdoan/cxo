package node

import (
	"errors"
	"net"
	"time"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/daemon/gnet"

	"github.com/skycoin/cxo/data"
)

var (
	ErrClosed   = errors.New("use of closed node")
	ErrNotFound = errors.New("not found")

	ErrManualDisconnect gnet.DisconnectReason = errors.New(
		"manual disconnect")
)

// A Config represents configurations of Node
type Config struct {
	gnet.Config

	// Known is a list of known addresses to subscribe to
	Known []string
	Debug bool
	Name  string
	Ping  time.Duration

	// RPC
	RPC        bool   // enable/disabel rpc
	RPCEvents  int    // evets channel size
	RPCAddress string // listen on (empty to auto)
	RPCMax     int    // max RPC connections
}

func NewConfig() Config {
	return Config{
		Config: gnet.NewConfig(),
		Ping:   5 * time.Second,

		RPC:        true,
		RPCEvents:  10,
		RPCAddress: "",
		RPCMax:     10,
	}
}

type Node struct {
	Logger
	conf Config
	db   *data.DB
	pool *gnet.ConnectionPool
	rpc  *RPC
	rpce chan func(*Node)
	quit chan struct{}
	done chan struct{}
}

func NewNode(conf Config, db *data.DB) *Node {
	if db == nil {
		panic("NewNode has got nil database")
	}
	if conf.Name == "" {
		conf.Name = "node"
	}
	if conf.Debug == false {
		gnet.DebugPrint = false
	}
	return &Node{
		Logger: NewLogger("["+conf.Name+"] ", conf.Debug),
		conf:   conf,
		db:     db,
	}
}

func (n *Node) Start() (err error) {
	n.Debug("[DBG] starting node")
	n.quit, n.done = make(chan struct{}), make(chan struct{})
	n.conf.ConnectCallback = func(gc *gnet.Connection, outgoing bool) {
		n.Debug("[DBG] got new connection ", gc.Addr())
		if !outgoing {
			n.sendEverythingWeHave(gc)
		}
	}
	n.pool = gnet.NewConnectionPool(n.conf.Config, n)
	if err = n.pool.StartListen(); err != nil {
		return
	}
	var addr net.Addr
	if addr, err = n.pool.ListeningAddress(); err != nil {
		n.Panic("[CRITICAL] can't obtain lisening address: ", err)
		return // never happens
	} else {
		n.Print("[INF] listening on ", addr)
	}
	if n.conf.RPC {
		n.rpc = NewRPC(n.conf.RPCEvents, n)
		if err = n.rpc.start(n.conf.RPCAddress, n.conf.RPCMax); err != nil {
			n.Print("[ERR] failed to start RPC: ", err)
		} else {
			n.rpce = n.rpc.events
		}
	}
	go n.pool.AcceptConnections()
	go n.handle(n.quit, n.done)
	go n.connectToKnown(n.quit)
	return
}

func (n *Node) sendEverythingWeHave(gc *gnet.Connection) {
	n.Debug("[DBG] send everything we have to ", gc.Addr())
	n.db.Range(func(k cipher.SHA256) {
		gc.ConnectionPool.SendMessage(gc, &Announce{k})
	})
}

func (n *Node) handle(quit, done chan struct{}) {
	n.Debug("[DBG] start handling events")
	var (
		tk   *time.Ticker
		ping <-chan time.Time
	)
	if n.conf.Ping > 0 {
		tk = time.NewTicker(n.conf.Ping)
		ping = tk.C
		defer tk.Stop()
	}
	defer close(done)
	defer n.pool.StopListen()
	for {
		select {
		case sr := <-n.pool.SendResults:
			if sr.Error != nil {
				n.Printf("[ERR] error sending %T to %s: %v",
					sr.Message,
					sr.Connection.Addr(),
					sr.Error)
			}
		case de := <-n.pool.DisconnectQueue:
			n.Debug("[DBG] disconnet %s because: ",
				n.pool.Pool[de.ConnId].Addr(),
				de.Reason)
			n.pool.HandleDisconnectEvent(de)
		case <-ping:
			n.Debug("[DBG] send pings")
			n.pool.BroadcastMessage(&Ping{})
		case rpce := <-n.rpce:
			rpce(n)
		case <-quit:
			return
		default:
			n.pool.HandleMessages()
		}
	}
}

func (n *Node) connectToKnown(quit chan struct{}) {
	n.Debug("[DBG] connecting to know hosts")
	for _, a := range n.conf.Known {
		n.Debug("[DBG] connecting to ", a)
		if _, err := n.pool.Connect(a); err != nil {
			n.Printf("[ERR] error connection to known host %s: %v", a, err)
		}
	}
}

func (n *Node) Close() {
	n.Debug("[DBG] cling node...")
	close(n.quit)
	if n.conf.RPC {
		n.rpc.close()
	}
	<-n.done
	n.Debug("[DBG] node was closed")
}
