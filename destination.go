package dendrite

import (
	"github.com/fizx/logs"
	"io"
	"net/url"
)

type Destinations []*Destination

type Destination struct {
	Encoder Encoder
	RW      io.ReadWriter
	sender  Sender
}

type Sender interface {
	io.Closer
	Send(Record) error
}

type SendDriver interface {
	Open(*url.URL) (Sender, error)
}

var sendDrivers = map[string]SendDriver{}

func RegisterSendDriver(name string, driver SendDriver) {
	if driver == nil {
		panic("dendrite RegisterSend driver is nil.")
	}
	if _, dup := sendDrivers[name]; dup {
		panic("dendrite RegisterSend called twice for driver " + name)
	}
	sendDrivers[name] = driver
}

func (dests *Destinations) Close() error {
	var err error
	for _, dest := range *dests {
		if e := dest.Close(); e != nil {
			logs.Warn(e)
			if err == nil {
				err = e
			}
		}
	}
	return err
}

func (dests *Destinations) Consume(ch chan Record, finished chan bool) {
	if len(*dests) == 0 {
		logs.Warn("No destinations specified")
	}
	for {
		rec := <-ch

		if rec == nil {
			break
		} else {
			for _, dest := range *dests {
				dest.Send(rec)
			}
		}
	}
	logs.Info("Finished consuming log records")
	finished <- true
}

func (dests *Destinations) Reader() io.Reader {
	var readers = make([]io.Reader, 0)
	for _, dest := range *dests {
		readers = append(readers, dest.RW)
	}
	return NewAnyReader(readers)
}

func NewDestinations() Destinations {
	return make([]*Destination, 0)
}

func NewDestination(config DestinationConfig) (*Destination, error) {
	var err error = nil
	dest := new(Destination)
	if driver, ok := sendDrivers[config.Url.Scheme]; ok {
		dest.sender, err = driver.Open(config.Url)
		if err != nil {
			return nil, err
		}
	} else {
		dest.RW, err = NewReadWriter(config.Url)
		if err != nil {
			return nil, err
		}
		dest.Encoder, err = NewEncoder(config.Url)
		if err != nil {
			return nil, err
		}
	}
	return dest, nil
}

func (dest *Destination) Close() error {
	if dest.sender != nil {
		return dest.sender.Close()
	} else if closer, ok := dest.RW.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

func (dest *Destination) Send(rec Record) error {
	if dest.sender != nil {
		return dest.sender.Send(rec)
	}
	dest.Encoder.Encode(rec, dest.RW)
	return nil
}
