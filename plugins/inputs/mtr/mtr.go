package mtr

import (
	"context"
	"strings"
	"time"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/internal"
	"github.com/influxdata/telegraf/plugins/inputs"
	"github.com/tonobo/mtr/pkg/mtr"
)

type MTR struct {
	URLs           []string
	Timeout        internal.Duration
	SrcAddr        string
	HopsMax        int
	HopsSleep      internal.Duration
	HopsMaxUnknown int
	RingBufferSize int
	PtrLookup      bool
}

var sampleConfig = `
  urls = ["domain.tld", "192.168.1.1", "fd79::1"]
  timeout = "40s"
`

var defaultTimeout = 1 * time.Second

// SampleConfig returns sample configuration message
func (m *MTR) SampleConfig() string {
	return sampleConfig
}

// Description returns description
func (m *MTR) Description() string {
	return `Traceroute hosts`
}

// Gather reads stats from all configured servers accumulates stats
func (m *MTR) Gather(acc telegraf.Accumulator) error {
	ctx := context.Background()

	if m.Timeout.Duration < 1*time.Second {
		m.Timeout.Duration = defaultTimeout
	}
	if m.HopsMax == 0 {
		m.HopsMax = 255
	}

	if m.HopsMaxUnknown == 0 {
		m.HopsMaxUnknown = 255
	}


	if m.RingBufferSize == 0 {
		m.RingBufferSize = 255
	}

	ctx, cancel := context.WithTimeout(ctx, m.Timeout.Duration)
	defer cancel()

	ch := make(chan struct{})

	go func(){
		for { <-ch }
	}()

	for _, url := range m.URLs {
		mtr, _, err := mtr.NewMTR(url, m.SrcAddr, m.Timeout.Duration, 0, m.HopsSleep.Duration,
			m.HopsMax, m.HopsMaxUnknown, m.RingBufferSize, m.PtrLookup)
		if err != nil {
			return err
		}
		mtr.Run(ch, 1)
		for hop, stat := range mtr.Statistic {
			tags := map[string]string{
				"url": url,
				"hop": strings.Join(stat.Targets, ","),
			}
			fields := map[string]interface{}{
				"hop": hop,
				"sent": stat.Sent,
				"ttl": stat.TTL,
				"last": stat.Last.Elapsed.Seconds() * 1000,
				"best": stat.Best.Elapsed.Seconds() * 1000,
				"worst": stat.Worst.Elapsed.Seconds() * 1000,
				"lost": stat.Loss(),
				"avg": stat.Avg(),
				"stdev": stat.Stdev(),
			}
			acc.AddFields("mtr", fields, tags)
		}
	}
	return nil
}

func init() {
	inputs.Add("mtr", func() telegraf.Input {
		return &MTR{}
	})
}
