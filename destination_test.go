package dendrite

import (
	"net/url"
	"sync/atomic"
	"testing"
)

var chanmux chan Record
var chanmuxcount int32

type chanmuxdriver struct{}

func (*chanmuxdriver) Open(u *url.URL) (Sender, error) {
	if atomic.AddInt32(&chanmuxcount, 1) == 1 {
		chanmux = make(chan Record)
	}
	return new(chanmuxsender), nil
}

type chanmuxsender struct{}

func (*chanmuxsender) Close() error {
	if atomic.AddInt32(&chanmuxcount, -1) == 0 {
		close(chanmux)
	}
	return nil
}

func (*chanmuxsender) Send(rec Record) error {
	chanmux <- rec
	return nil
}

func init() {
	RegisterSendDriver("chanmux", new(chanmuxdriver))
}

func TestRegisterSender(t *testing.T) {
	u, _ := url.Parse("chanmux://")
	config := DestinationConfig{Name: "test", Url: u}
	dest, err := NewDestination(config)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		dests := NewDestinations()
		defer dests.Close()
		dests = append(dests, dest)
		ch := make(chan Record, 2)
		ch <- Record{"key": Column{Value: "value"}}
		close(ch)
		dests.Consume(ch, make(chan bool, 1))
	}()
	if rec, ok := <-chanmux; !ok {
		t.Fatal("received no records at all")
	} else if rec["key"].Value != "value" {
		t.Fatal("received wrong record")
	}
	if _, ok := <-chanmux; ok {
		t.Fatal("received too many records")
	}
}
