// Copyright 2023 Grafana Labs
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// TODO: add go build tag

package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/alecthomas/kingpin/v2"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/ibm-db2-prometheus-exporter/collector"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/promlog/flag"
	"github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"
	webflag "github.com/prometheus/exporter-toolkit/web/kingpinflag"
)

var (
	webConfig  = webflag.AddFlags(kingpin.CommandLine, ":9953")
	metricPath = kingpin.Flag("web.telemetry-path", "Path under which to expose metrics.").Default("/metrics").Envar("IBM_DB2_EXPORTER_WEB_TELEMETRY_PATH").String()
	dsn        = kingpin.Flag("dsn", "The connection string (data source name) to use to connect to the database when querying metrics.").Envar("IBM_DB2_EXPORTER_DSN").Required().String()

	// database  = kingpin.Flag("database", "The name of the database to connect to when querying metrics.").Envar("IBM_DB2_EXPORTER_DATABASE").Required().String()
	// host      = kingpin.Flag("host", "The name of the host used to connect to the database when querying metrics.").Envar("IBM_DB2_EXPORTER_HOST").Required().String()
	// port      = kingpin.Flag("port", "The number of the port used to connect to the database when querying metrics.").Envar("IBM_DB2_EXPORTER_PORT").Required().String()
	// uid       = kingpin.Flag("username", "The username or uid used to connect to the database when querying metrics").Envar("IBM_DB2_EXPORTER_USERNAME").Required().String()
	// pwd       = kingpin.Flag("password", "The password used to connect to the database when querying metrics").Envar("IBM_DB2_EXPORTER_PASSWORD").Required().String()
	// dsnaddons = kingpin.Flag("dsnaddons", "Additional DSN fields used to connect to the database when querying metrics").Envar("IBM_DB2_EXPORTER_PASSWORD").String()
)

const (
	// The name of the exporter.
	exporterName    = "ibm_db2_exporter"
	landingPageHtml = `
	<html>
		<head><title>IBM DB2 exporter</title></head>
		<body>
			<h1>IBM DB2 exporter</h1>
			<p><a href='%s'>Metrics</a></p>
		</body>
	</html>`
)

func main() {
	kingpin.Version(version.Print(exporterName))

	promlogConfig := &promlog.Config{}

	flag.AddFlags(kingpin.CommandLine, promlogConfig)
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	logger := promlog.New(promlogConfig)

	// Construct the collector, using the flags for configuration
	c := &collector.Config{}

	if err := c.Validate(); err != nil {
		level.Error(logger).Log("msg", "Configuration is invalid.", "err", err)
		os.Exit(1)
	}

	col := collector.NewCollector(logger, c)

	// Register collector with prometheus client library
	prometheus.MustRegister(version.NewCollector(exporterName))
	prometheus.MustRegister(col)

	serveMetrics(logger)
}

func serveMetrics(logger log.Logger) {

	landingPage := []byte(fmt.Sprintf(landingPageHtml, *metricPath))

	http.Handle(*metricPath, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=UTF-8") // nolint: errcheck
		w.Write(landingPage)                                       // nolint: errcheck
	})

	srv := &http.Server{}
	if err := web.ListenAndServe(srv, webConfig, logger); err != nil {
		level.Error(logger).Log("msg", "Error running HTTP server", "err", err)
		os.Exit(1)
	}
}
