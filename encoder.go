package dendrite

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"
)

type Encoder interface {
	Encode(out map[string]Column, writer io.Writer)
}

type JsonEncoder struct{}
type StatsdEncoder struct{}
type RawStringEncoder struct{}

var encoders = make(map[string]Encoder)

func RegisterEncoder(name string, encoder Encoder) {
	if encoder == nil {
		panic("dendrite RegisterEncoder encoder is nil.")
	}
	if _, dup := encoders[name]; dup {
		panic("dendrite RegisterEncoder called twice for encoder " + name)
	}
	encoders[name] = encoder
}

func init() {
	RegisterEncoder("json", new(JsonEncoder))
	RegisterEncoder("statsd", new(StatsdEncoder))
}

func NewEncoder(u *url.URL) (Encoder, error) {
	a := strings.Split(u.Scheme, "+")
	if len(a) > 1 {
		if encoder, ok := encoders[a[len(a)-1]]; ok {
			return encoder, nil
		}
	}
	return new(RawStringEncoder), nil
}

func (*RawStringEncoder) Encode(out map[string]Column, writer io.Writer) {
	for _, v := range out {
		if v.Type == String {
			writer.Write([]byte(v.Value.(string) + "\n"))
		}
	}
}

func (*JsonEncoder) Encode(out map[string]Column, writer io.Writer) {
	stripped := make(map[string]interface{})
	for k, v := range out {
		stripped[k] = v.Value
	}
	bytes, err := json.Marshal(stripped)
	if err != nil {
		panic(err)
	}
	bytes = append(bytes, '\n')
	writer.Write(bytes)
}

func (*StatsdEncoder) Encode(out map[string]Column, writer io.Writer) {
	for k, v := range out {
		switch v.Type {
		case Gauge:
			writer.Write([]byte(fmt.Sprintf("%s:%d|g", k, v.Value)))
		case Metric:
			writer.Write([]byte(fmt.Sprintf("%s:%d|m", k, v.Value)))
		case Counter:
			writer.Write([]byte(fmt.Sprintf("%s:%d|c", k, v.Value)))
		}
	}
}
