package main

import (
	"fmt"
	"iter"
	"log/slog"
	"math/rand"
	"slices"
	"sync"
	"time"
)

type event struct {
	processID string // node id where this event was registered
	timestamp int
	payload   string
}

// before returns true if a comes before b.
//
// Uses tie-breakers to ensure a total ordering of events with deterministic results.
func before(a, b event) bool {
	if a.timestamp != b.timestamp {
		// no need for tie-breakers
		return a.timestamp < b.timestamp
	}
	return a.processID < b.processID
}

// command tells a node what action to perform
type command struct {
	action string // "send" or "stop"
	dest   *node  // destination node for send
}

type node struct {
	procID      string
	logicalTime int
	inbox       chan event   // incoming messages from other nodes
	commands    chan command // commands to this node's event loop
	eventLog    []event
}

func newNode(id string) *node {
	return &node{
		procID:   id,
		inbox:    make(chan event, 10),
		commands: make(chan command),
		eventLog: make([]event, 0),
	}
}

// send is called within the node's own event loop - no concurrency issues
func (n *node) send(dest *node) {
	n.logicalTime++
	ev := event{
		processID: n.procID,
		timestamp: n.logicalTime,
		payload:   fmt.Sprintf("[%s] send -> %s", n.procID, dest.procID),
	}
	n.eventLog = append(n.eventLog, ev)
	slog.Info("sending", slog.String("from", n.procID), slog.String("to", dest.procID), slog.Int("ts", ev.timestamp))

	// send message to destination's inbox
	dest.inbox <- event{
		processID: n.procID,
		timestamp: n.logicalTime,
		payload:   fmt.Sprintf("msg from %s", n.procID),
	}
}

// receive is called within the node's own event loop
func (n *node) receive(ev event) {
	n.logicalTime = max(n.logicalTime, ev.timestamp) + 1
	logEv := event{
		processID: n.procID,
		timestamp: n.logicalTime,
		payload:   fmt.Sprintf("[%s] recv <- %s", n.procID, ev.payload),
	}
	n.eventLog = append(n.eventLog, logEv)
	slog.Info("received", slog.String("node", n.procID), slog.Int("ts", n.logicalTime))
}

// run is the node's single-threaded event loop
// all state mutations happen here - no concurrent access
func (n *node) run(wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case ev := <-n.inbox:
			n.receive(ev)
		case cmd := <-n.commands:
			switch cmd.action {
			case "send":
				n.send(cmd.dest)
			case "stop":
				// drain any remaining messages before stopping
				for {
					select {
					case ev := <-n.inbox:
						n.receive(ev)
					default:
						return
					}
				}
			}
		}
	}
}

func (n *node) events() iter.Seq[event] {
	return slices.Values(n.eventLog)
}

func (n *node) totalEvents() int {
	return len(n.eventLog)
}

func main() {
	const totalNodes = 5
	nodes := make([]*node, totalNodes)
	var wg sync.WaitGroup

	// create and start all nodes
	for i := 0; i < totalNodes; i++ {
		nodes[i] = newNode(fmt.Sprintf("node-%d", i+1))
		wg.Add(1)
		go nodes[i].run(&wg)
	}

	// simulate message passing
	// commands are processed by each node's own event loop
	const totalMessages = 20
	for i := 0; i < totalMessages; i++ {
		sender := rand.Intn(totalNodes)
		receiver := (sender + 1 + rand.Intn(totalNodes-1)) % totalNodes // pick different node

		nodes[sender].commands <- command{
			action: "send",
			dest:   nodes[receiver],
		}

		// small delay to let messages propagate (simulates network latency)
		time.Sleep(10 * time.Millisecond)
	}

	// give time for final messages to be processed
	time.Sleep(50 * time.Millisecond)

	// stop all nodes
	for _, n := range nodes {
		n.commands <- command{action: "stop"}
	}
	wg.Wait()

	// print event logs
	fmt.Println()
	totalEvents := 0
	eventLog := make([]event, 0)
	for _, n := range nodes {
		slog.Info("event log", slog.String("node", n.procID))
		totalEvents += n.totalEvents()
		eventLog = slices.AppendSeq(eventLog, n.events())
		for ev := range n.events() {
			slog.Info("  event", slog.String("payload", ev.payload), slog.Int("ts", ev.timestamp))
		}
		fmt.Println()
	}

	// print global event log
	fmt.Println()
	slices.SortFunc(eventLog, func(a, b event) int {
		if before(a, b) {
			return -1
		} else if before(b, a) {
			return 1
		}
		return 0
	})
	slog.Info("global event log", slog.Int("total events", totalEvents))
	for _, ev := range eventLog {
		slog.Info("event", slog.String("payload", ev.payload), slog.Int("ts", ev.timestamp))
	}
}
