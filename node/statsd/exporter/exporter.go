// Copyright (C) 2014-2018 Wutong Co., Ltd.
// WUTONG, Application Management Platform

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version. For any non-GPL usage of Wutong,
// one or multiple Commercial Licenses authorized by Wutong Co., Ltd.
// must be obtained first.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package exporter

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"io"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/prometheus/common/model"
	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/node/statsd/prometheus"
)

const (
	defaultHelp = "Metric autogenerated by statsd_exporter."
	regErrF     = "A change of configuration created inconsistent metrics for " +
		"%q. You have to restart the statsd_exporter, and you should " +
		"consider the effects on your monitoring setup. Error: %s"
)

var (
	illegalCharsRE = regexp.MustCompile(`[^a-zA-Z0-9_]`)

	hash   = fnv.New64a()
	strBuf bytes.Buffer // Used for hashing.
	intBuf = make([]byte, 8)
)

// hashNameAndLabels returns a hash value of the provided name string and all
// the label names and values in the provided labels map.
//
// Not safe for concurrent use! (Uses a shared buffer and hasher to save on
// allocations.)
func hashNameAndLabels(name string, labels prometheus.Labels) uint64 {
	hash.Reset()
	strBuf.Reset()
	strBuf.WriteString(name)
	hash.Write(strBuf.Bytes())
	binary.BigEndian.PutUint64(intBuf, model.LabelsToSignature(labels))
	hash.Write(intBuf)
	return hash.Sum64()
}

type CounterContainer struct {
	Elements map[uint64]prometheus.Counter
	Register prometheus.Registerer
}

func NewCounterContainer(Register prometheus.Registerer) *CounterContainer {
	return &CounterContainer{
		Elements: make(map[uint64]prometheus.Counter),
		Register: Register,
	}
}

func (c *CounterContainer) Get(metricName string, labels prometheus.Labels, help string) (prometheus.Counter, error) {
	hash := hashNameAndLabels(metricName, labels)
	counter, ok := c.Elements[hash]
	if !ok {
		counter = prometheus.NewCounter(prometheus.CounterOpts{
			Name:        metricName,
			Help:        help,
			ConstLabels: labels,
		})
		if err := c.Register.Register(counter); err != nil {
			return nil, err
		}
		c.Elements[hash] = counter
	}
	return counter, nil
}

type GaugeContainer struct {
	Elements map[uint64]prometheus.Gauge
	Register prometheus.Registerer
}

func NewGaugeContainer(Register prometheus.Registerer) *GaugeContainer {
	return &GaugeContainer{
		Elements: make(map[uint64]prometheus.Gauge),
		Register: Register,
	}
}

func (c *GaugeContainer) Get(metricName string, labels prometheus.Labels, help string) (prometheus.Gauge, error) {
	hash := hashNameAndLabels(metricName, labels)
	gauge, ok := c.Elements[hash]
	if !ok {
		gauge = prometheus.NewGauge(prometheus.GaugeOpts{
			Name:        metricName,
			Help:        help,
			ConstLabels: labels,
		})
		if err := c.Register.Register(gauge); err != nil {
			return nil, err
		}
		c.Elements[hash] = gauge
	}
	return gauge, nil
}

type SummaryContainer struct {
	Elements map[uint64]prometheus.Summary
	Register prometheus.Registerer
}

func NewSummaryContainer(Register prometheus.Registerer) *SummaryContainer {
	return &SummaryContainer{
		Elements: make(map[uint64]prometheus.Summary),
		Register: Register,
	}
}

func (c *SummaryContainer) Get(metricName string, labels prometheus.Labels, help string) (prometheus.Summary, error) {
	hash := hashNameAndLabels(metricName, labels)
	summary, ok := c.Elements[hash]
	if !ok {
		summary = prometheus.NewSummary(
			prometheus.SummaryOpts{
				Name:        metricName,
				Help:        help,
				ConstLabels: labels,
			})
		if err := c.Register.Register(summary); err != nil {
			return nil, err
		}
		c.Elements[hash] = summary
	}
	return summary, nil
}

type HistogramContainer struct {
	Elements map[uint64]prometheus.Histogram
	mapper   *MetricMapper
	Register prometheus.Registerer
}

func NewHistogramContainer(mapper *MetricMapper, Register prometheus.Registerer) *HistogramContainer {
	return &HistogramContainer{
		Elements: make(map[uint64]prometheus.Histogram),
		mapper:   mapper,
		Register: Register,
	}
}

func (c *HistogramContainer) Get(metricName string, labels prometheus.Labels, help string, mapping *metricMapping) (prometheus.Histogram, error) {
	hash := hashNameAndLabels(metricName, labels)
	histogram, ok := c.Elements[hash]
	if !ok {
		buckets := c.mapper.Defaults.Buckets
		if mapping != nil && mapping.Buckets != nil && len(mapping.Buckets) > 0 {
			buckets = mapping.Buckets
		}
		histogram = prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:        metricName,
				Help:        help,
				ConstLabels: labels,
				Buckets:     buckets,
			})
		c.Elements[hash] = histogram
		if err := c.Register.Register(histogram); err != nil {
			return nil, err
		}
	}
	return histogram, nil
}

type Event interface {
	MetricName() string
	Value() float64
	Labels() map[string]string
}

type CounterEvent struct {
	metricName string
	value      float64
	labels     map[string]string
}

func (c *CounterEvent) MetricName() string        { return c.metricName }
func (c *CounterEvent) Value() float64            { return c.value }
func (c *CounterEvent) Labels() map[string]string { return c.labels }

type GaugeEvent struct {
	metricName string
	value      float64
	relative   bool
	labels     map[string]string
}

func (g *GaugeEvent) MetricName() string        { return g.metricName }
func (g *GaugeEvent) Value() float64            { return g.value }
func (c *GaugeEvent) Labels() map[string]string { return c.labels }

type TimerEvent struct {
	metricName string
	value      float64
	labels     map[string]string
}

func (t *TimerEvent) MetricName() string        { return t.metricName }
func (t *TimerEvent) Value() float64            { return t.value }
func (c *TimerEvent) Labels() map[string]string { return c.labels }

type Events []Event

type Exporter struct {
	Counters   *CounterContainer
	Gauges     *GaugeContainer
	Summaries  *SummaryContainer
	Histograms *HistogramContainer
	mapper     *MetricMapper
	vitality   int64
}

func escapeMetricName(metricName string) string {
	// If a metric starts with a digit, prepend an underscore.
	if metricName[0] >= '0' && metricName[0] <= '9' {
		metricName = "_" + metricName
	}

	// Replace all illegal metric chars with underscores.
	metricName = illegalCharsRE.ReplaceAllString(metricName, "_")
	return metricName
}

func (b *Exporter) Listen(e <-chan Events) {
	for {
		events, ok := <-e
		if !ok {
			logrus.Debug("Channel is closed. Break out of Exporter.Listener.")
			return
		}
		for _, event := range events {
			var help string
			metricName := ""
			prometheusLabels := event.Labels()
			timestamp := time.Now().Unix()

			mapping, labels, present := b.mapper.getMapping(event.MetricName())
			if mapping == nil {
				mapping = &metricMapping{}
			}
			if mapping.HelpText == "" {
				help = defaultHelp
			} else {
				help = mapping.HelpText
			}
			if present {
				metricName = mapping.Name
				for label, value := range labels {
					prometheusLabels[label] = value
				}
			} else {
				eventsUnmapped.Inc()
				metricName = escapeMetricName(event.MetricName())
			}

			switch ev := event.(type) {
			case *CounterEvent:
				// We don't accept negative values for counters. Incrementing the counter with a negative number
				// will cause the exporter to panic. Instead we will warn and continue to the next event.
				if event.Value() < 0.0 {
					logrus.Debugf("Counter %q is: '%f' (counter must be non-negative value)", metricName, event.Value())
					eventStats.WithLabelValues("illegal_negative_counter").Inc()
					continue
				}

				counter, err := b.Counters.Get(
					metricName,
					prometheusLabels,
					help,
				)
				if err == nil {
					counter.Add(event.Value())
					counter.SetTimestamp(timestamp)
					eventStats.WithLabelValues("counter").Inc()
				} else {
					logrus.Debugf(regErrF, metricName, err)
					conflictingEventStats.WithLabelValues("counter").Inc()
				}

			case *GaugeEvent:
				gauge, err := b.Gauges.Get(
					metricName,
					prometheusLabels,
					help,
				)

				if err == nil {
					if ev.relative {
						gauge.Add(event.Value())
					} else {
						gauge.Set(event.Value())
					}

					gauge.SetTimestamp(timestamp)
					eventStats.WithLabelValues("gauge").Inc()
				} else {
					logrus.Debugf(regErrF, metricName, err)
					conflictingEventStats.WithLabelValues("gauge").Inc()
				}

			case *TimerEvent:
				t := timerTypeDefault
				if mapping != nil {
					t = mapping.TimerType
				}
				if t == timerTypeDefault {
					t = b.mapper.Defaults.TimerType
				}

				switch t {
				case timerTypeHistogram:
					histogram, err := b.Histograms.Get(
						metricName,
						prometheusLabels,
						help,
						mapping,
					)
					if err == nil {
						histogram.Observe(event.Value() / 1000) // prometheus presumes seconds, statsd millisecond
						histogram.SetTimestamp(timestamp)
						eventStats.WithLabelValues("timer").Inc()
					} else {
						logrus.Debugf(regErrF, metricName, err)
						conflictingEventStats.WithLabelValues("timer").Inc()
					}

				case timerTypeDefault, timerTypeSummary:
					summary, err := b.Summaries.Get(
						metricName,
						prometheusLabels,
						help,
					)
					if err == nil {
						summary.Observe(event.Value())
						summary.SetTimestamp(timestamp)
						eventStats.WithLabelValues("timer").Inc()
					} else {
						logrus.Debugf(regErrF, metricName, err)
						conflictingEventStats.WithLabelValues("timer").Inc()
					}

				default:
					panic(fmt.Sprintf("unknown timer type '%s'", t))
				}

			default:
				logrus.Debugln("Unsupported event type")
				eventStats.WithLabelValues("illegal").Inc()
			}
		}
	}
}

// GCollector 循环检查Exporter对象中的性能指标数据是否有过期，有则清除
func (b *Exporter) GCollector() {
	var HP = b.vitality
	timer := time.NewTicker(time.Second * 10)
	defer timer.Stop()
	for {
		<-timer.C
		currentTime := time.Now().Unix()
		oldCounters := len(b.Counters.Elements)
		oldGauges := len(b.Gauges.Elements)
		oldHistograms := len(b.Histograms.Elements)
		oldSummaries := len(b.Summaries.Elements)
		for k, v := range b.Counters.Elements {
			oldTime := v.GetTimestamp()
			if (currentTime - oldTime) > HP {
				delete(b.Counters.Elements, k)
				b.Counters.Register.Unregister(v)
			}
		}
		for k, v := range b.Gauges.Elements {
			oldTime := v.GetTimestamp()
			if (currentTime - oldTime) > HP {
				delete(b.Gauges.Elements, k)
				b.Gauges.Register.Unregister(v)
			}
		}
		for k, v := range b.Histograms.Elements {
			oldTime := v.GetTimestamp()
			if (currentTime - oldTime) > HP {
				delete(b.Histograms.Elements, k)
				b.Histograms.Register.Unregister(v)
			}
		}
		for k, v := range b.Summaries.Elements {
			oldTime := v.GetTimestamp()
			if (currentTime - oldTime) > HP {
				delete(b.Summaries.Elements, k)
				b.Summaries.Register.Unregister(v)
			}
		}
		logrus.Debugf("current amount for Counters: %v => %v", oldCounters, len(b.Counters.Elements))
		logrus.Debugf("current amount for Gauges: %v => %v", oldGauges, len(b.Gauges.Elements))
		logrus.Debugf("current amount for Histograms: %v => %v", oldHistograms, len(b.Histograms.Elements))
		logrus.Debugf("current amount for Summaries: %v => %v", oldSummaries, len(b.Summaries.Elements))
	}
}

// NewExporter new exporter
func NewExporter(mapper *MetricMapper, Register prometheus.Registerer) *Exporter {
	return &Exporter{
		Counters:   NewCounterContainer(Register),
		Gauges:     NewGaugeContainer(Register),
		Summaries:  NewSummaryContainer(Register),
		Histograms: NewHistogramContainer(mapper, Register),
		mapper:     mapper,
		vitality:   20,
	}
}

func buildEvent(statType, metric string, value float64, relative bool, labels map[string]string) (Event, error) {
	switch statType {
	case "c":
		return &CounterEvent{
			metricName: metric,
			value:      float64(value),
			labels:     labels,
		}, nil
	case "g":
		return &GaugeEvent{
			metricName: metric,
			value:      float64(value),
			relative:   relative,
			labels:     labels,
		}, nil
	case "ms", "h":
		return &TimerEvent{
			metricName: metric,
			value:      float64(value),
			labels:     labels,
		}, nil
	case "s":
		return nil, fmt.Errorf("No support for StatsD sets")
	default:
		return nil, fmt.Errorf("Bad stat type %s", statType)
	}
}

func parseDogStatsDTagsToLabels(component string) map[string]string {
	labels := map[string]string{}
	tagsReceived.Inc()
	tags := strings.Split(component, ",")
	for _, t := range tags {
		t = strings.TrimPrefix(t, "#")
		kv := strings.SplitN(t, ":", 2)

		if len(kv) < 2 || len(kv[1]) == 0 {
			tagErrors.Inc()
			logrus.Debugf("Malformed or empty DogStatsD tag %s in component %s", t, component)
			continue
		}

		labels[escapeMetricName(kv[0])] = kv[1]
	}
	return labels
}

func lineToEvents(line string) Events {
	events := Events{}
	if line == "" {
		return events
	}

	elements := strings.SplitN(line, ":", 2)
	if len(elements) < 2 || len(elements[0]) == 0 || !utf8.ValidString(line) {
		sampleErrors.WithLabelValues("malformed_line").Inc()
		logrus.Debugln("Bad line from StatsD:", line)
		return events
	}
	metric := elements[0]
	var samples []string
	if strings.Contains(elements[1], "|#") {
		// using datadog extensions, disable multi-metrics
		samples = elements[1:]
	} else {
		samples = strings.Split(elements[1], ":")
	}
samples:
	for _, sample := range samples {
		samplesReceived.Inc()
		components := strings.Split(sample, "|")
		if len(components) < 2 || len(components) > 4 {
			sampleErrors.WithLabelValues("malformed_component").Inc()
			logrus.Debugln("Bad component on line:", line)
			continue
		}
		valueStr, statType := components[0], components[1]

		var relative = false
		if strings.Index(valueStr, "+") == 0 || strings.Index(valueStr, "-") == 0 {
			relative = true
		}

		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			logrus.Debugf("Bad value %s on line: %s", valueStr, line)
			sampleErrors.WithLabelValues("malformed_value").Inc()
			continue
		}

		multiplyEvents := 1
		labels := map[string]string{}
		if len(components) >= 3 {
			for _, component := range components[2:] {
				if len(component) == 0 {
					logrus.Debugln("Empty component on line: ", line)
					sampleErrors.WithLabelValues("malformed_component").Inc()
					continue samples
				}
			}

			for _, component := range components[2:] {
				switch component[0] {
				case '@':
					if statType != "c" && statType != "ms" {
						logrus.Debugln("Illegal sampling factor for non-counter metric on line", line)
						sampleErrors.WithLabelValues("illegal_sample_factor").Inc()
						continue
					}
					samplingFactor, err := strconv.ParseFloat(component[1:], 64)
					if err != nil {
						logrus.Debugf("Invalid sampling factor %s on line %s", component[1:], line)
						sampleErrors.WithLabelValues("invalid_sample_factor").Inc()
					}
					if samplingFactor == 0 {
						samplingFactor = 1
					}

					if statType == "c" {
						value /= samplingFactor
					} else if statType == "ms" {
						multiplyEvents = int(1 / samplingFactor)
					}
				case '#':
					labels = parseDogStatsDTagsToLabels(component)
				default:
					logrus.Debugf("Invalid sampling factor or tag section %s on line %s", components[2], line)
					sampleErrors.WithLabelValues("invalid_sample_factor").Inc()
					continue
				}
			}
		}

		for i := 0; i < multiplyEvents; i++ {
			event, err := buildEvent(statType, metric, value, relative, labels)
			if err != nil {
				logrus.Debugf("Error building event on line %s: %s", line, err)
				sampleErrors.WithLabelValues("illegal_event").Inc()
				continue
			}
			events = append(events, event)
		}
	}
	return events
}

type StatsDUDPListener struct {
	Conn *net.UDPConn
}

func (l *StatsDUDPListener) Listen(e chan<- Events) {
	buf := make([]byte, 65535)
	for {
		n, _, err := l.Conn.ReadFromUDP(buf)
		if err != nil {
			logrus.Fatal(err)
		}
		l.handlePacket(buf[0:n], e)
	}
}

func (l *StatsDUDPListener) handlePacket(packet []byte, e chan<- Events) {
	udpPackets.Inc()
	lines := strings.Split(string(packet), "\n")
	events := Events{}
	for _, line := range lines {
		linesReceived.Inc()
		events = append(events, lineToEvents(line)...)
	}
	e <- events
}

type StatsDTCPListener struct {
	Conn *net.TCPListener
}

func (l *StatsDTCPListener) Listen(e chan<- Events) {
	defer l.Conn.Close()
	for {
		c, err := l.Conn.AcceptTCP()
		if err != nil {
			logrus.Fatalf("AcceptTCP failed: %v", err)
		}
		go l.handleConn(c, e)
	}
}

func (l *StatsDTCPListener) handleConn(c *net.TCPConn, e chan<- Events) {
	defer c.Close()

	tcpConnections.Inc()

	r := bufio.NewReader(c)
	for {
		line, isPrefix, err := r.ReadLine()
		if err != nil {
			if err != io.EOF {
				tcpErrors.Inc()
				logrus.Debugf("Read %s failed: %v", c.RemoteAddr(), err)
			}
			break
		}
		if isPrefix {
			tcpLineTooLong.Inc()
			logrus.Debugf("Read %s failed: line too long", c.RemoteAddr())
			break
		}
		linesReceived.Inc()
		e <- lineToEvents(string(line))
	}
}
