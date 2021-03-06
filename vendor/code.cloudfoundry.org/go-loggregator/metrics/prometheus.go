package metrics

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

// The Registry keeps track of registered counters and gauges. Optionally, it can
// provide a server on a Prometheus-formatted endpoint.
type Registry struct {
	port        string
	defaultTags map[string]string
	loggr       *log.Logger
	registerer  prometheus.Registerer
}

// A cumulative metric that represents a single monotonically increasing counter
// whose value can only increase or be reset to zero on restart
type Counter interface {
	Add(float64)
}

// A single numerical value that can arbitrarily go up and down.
type Gauge interface {
	Add(float64)
	Set(float64)
}

// Registry will register the metrics route with the default http mux but will not
// start an http server. This is intentional so that we can combine metrics with
// other things like pprof into one server. To start a server
// just for metrics, use the WithServer RegistryOption
func NewRegistry(logger *log.Logger, opts ...RegistryOption) *Registry {
	pr := &Registry{
		loggr:       logger,
		defaultTags: make(map[string]string),
	}

	for _, o := range opts {
		o(pr)
	}

	registry := prometheus.NewRegistry()
	registerer := prometheus.WrapRegistererWith(pr.defaultTags, registry)
	pr.registerer = registerer

	registerer.MustRegister(prometheus.NewGoCollector())
	registerer.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))

	http.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{
		Registry: registerer,
	}))
	return pr
}

// Creates new counter. When a duplicate is registered, the Registry will return
// the previously created metric.
func (p *Registry) NewCounter(name string, opts ...MetricOption) Counter {
	opt := p.toPromOpt(name, "counter metric", opts...)
	c := prometheus.NewCounter(prometheus.CounterOpts(opt))
	return p.registerCollector(name, c).(Counter)
}

// Creates new gauge. When a duplicate is registered, the Registry will return
// the previously created metric.
func (p *Registry) NewGauge(name string, opts ...MetricOption) Gauge {
	opt := p.toPromOpt(name, "gauge metric", opts...)
	g := prometheus.NewGauge(prometheus.GaugeOpts(opt))
	return p.registerCollector(name, g).(Gauge)
}

func (p *Registry) registerCollector(name string, c prometheus.Collector) prometheus.Collector {
	err := p.registerer.Register(c)
	if err != nil {
		typ, ok := err.(prometheus.AlreadyRegisteredError)
		if !ok {
			p.loggr.Panicf("unable to create %s: %s", name, err)
		}

		return typ.ExistingCollector
	}

	return c
}

// Get the port of the running metrics server
func (p *Registry) Port() string {
	return fmt.Sprint(p.port)
}

func (p *Registry) toPromOpt(name, helpText string, mOpts ...MetricOption) prometheus.Opts {
	opt := prometheus.Opts{
		Name:        name,
		Help:        helpText,
		ConstLabels: make(map[string]string),
	}

	for _, o := range mOpts {
		o(&opt)
	}

	return opt
}

// Options for registry initialization
type RegistryOption func(r *Registry)

// Add Default tags to all gauges and counters created from this registry
func WithDefaultTags(tags map[string]string) RegistryOption {
	return func(r *Registry) {
		for k, v := range tags {
			r.defaultTags[k] = v
		}
	}
}

// Starts an http server on the given port to host metrics.
func WithServer(port int) RegistryOption {
	return func(r *Registry) {
		r.start(port)
	}
}

func (p *Registry) start(port int) {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	s := http.Server{
		Addr:         addr,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		p.loggr.Fatalf("Unable to setup metrics endpoint (%s): %s", addr, err)
	}
	p.loggr.Printf("Metrics endpoint is listening on %s", lis.Addr().String())

	parts := strings.Split(lis.Addr().String(), ":")
	p.port = parts[len(parts)-1]

	go s.Serve(lis)
}

// Options applied to metrics on creation
type MetricOption func(o *prometheus.Opts)

// Add these tags to the metrics
func WithMetricTags(tags map[string]string) MetricOption {
	return func(o *prometheus.Opts) {
		for k, v := range tags {
			o.ConstLabels[k] = v
		}
	}
}

// Add the passed help text to the created metric
func WithHelpText(helpText string) MetricOption {
	return func(o *prometheus.Opts) {
		o.Help = helpText
	}
}
