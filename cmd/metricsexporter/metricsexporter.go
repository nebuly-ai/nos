/*
 * Copyright 2023 nebuly.com
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	m "github.com/nebuly-ai/nos/cmd/metricsexporter/metrics"
	"io"
	"net/http"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/yaml"
)

func main() {
	// Setup CLI args
	var metricsFile string
	var metricsEndpoint string
	flag.StringVar(
		&metricsFile,
		"metrics-file",
		"",
		"Path to the JSON file containing the metrics to export.",
	)
	flag.StringVar(
		&metricsEndpoint,
		"metrics-endpoint",
		"",
		"HTTP endpoint to which send the metrics.",
	)
	opts := zap.Options{}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	// Setup context and logger
	ctx := ctrl.SetupSignalHandler()
	logger := log.FromContext(ctx)

	// Read metrics
	logger.Info("reading metrics file", "metricsFile", metricsFile)
	metricsFileBytes, err := os.ReadFile(metricsFile)
	if err != nil {
		logger.Error(err, "failed to read metrics file")
		os.Exit(0)
	}
	var metrics m.Metrics
	if err = yaml.Unmarshal(metricsFileBytes, &metrics); err != nil {
		logger.Error(err, "failed to unmarshal metrics file")
		os.Exit(0)
	}

	// Send metrics to Nebuly
	logger.Info("sending metrics to Nebuly", "metricsEndpoint", metricsEndpoint)
	body, err := json.Marshal(metrics)
	if err != nil {
		logger.Error(err, "failed to marshal metrics")
		os.Exit(0)
	}
	resp, err := http.Post(metricsEndpoint, "application/json", bytes.NewBuffer(body))
	if err != nil {
		logger.Error(err, "failed to send metrics")
		os.Exit(0)
	}
	respBody, _ := io.ReadAll(resp.Body)
	logger.Info(
		"metrics sent",
		"responseBody",
		string(respBody),
		"responseStatus",
		resp.Status,
	)
}
