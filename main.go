package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
)

// 全局声明 Gauge
var (
	oulaTotalGauge      prometheus.Gauge
	oulaTotalNewGauge   prometheus.Gauge
	zkworkGpuGauge      prometheus.Gauge
	cysicProofRateGauge prometheus.Gauge
)

func main() {
	// 定义命令行参数
	oulaLogPath := flag.String("oula-log", "", "Path to the oula log file")
	oulaNewLogPath := flag.String("oula-new-log", "", "Path to the new version oula log file")
	zkworkLogPath := flag.String("zkwork-log", "", "Path to the zkwork log file")
	cysicLogPath := flag.String("cysic-log", "", "Path to the cysic log file")
	pushgatewayURL := flag.String("pushgateway-url", "http://localhost:9091", "Pushgateway URL")
	jobName := flag.String("job-name", "log_monitor", "Job name for Pushgateway")
	instanceName := flag.String("instance-name", "instance1", "Instance name for Pushgateway")

	flag.Parse()

	// 初始化 Gauge
	initMetrics(*instanceName)

	// 启动日志监控
	if *oulaLogPath != "" {
		go monitorOulaLog(*oulaLogPath, *pushgatewayURL, *jobName, *instanceName, "v1", oulaTotalGauge)
	}

	if *oulaNewLogPath != "" {
		go monitorOulaLog(*oulaNewLogPath, *pushgatewayURL, *jobName, *instanceName, "v2", oulaTotalNewGauge)
	}

	if *zkworkLogPath != "" {
		go monitorZkworkLog(*zkworkLogPath, *pushgatewayURL, *jobName, *instanceName)
	}

	if *cysicLogPath != "" {
		go monitorCysicLog(*cysicLogPath, *pushgatewayURL, *jobName, *instanceName)
	}

	// 保持程序运行
	select {}
}

func initMetrics(instanceName string) {
	oulaTotalGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        "oula_total",
		Help:        "Total value from oula log",
		ConstLabels: prometheus.Labels{"instance": instanceName, "version": "v1"},
	})

	oulaTotalNewGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        "oula_total",
		Help:        "Total value from new version oula log",
		ConstLabels: prometheus.Labels{"instance": instanceName, "version": "v2"},
	})

	zkworkGpuGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        "zkwork_gpu",
		Help:        "GPU value from zkwork log",
		ConstLabels: prometheus.Labels{"instance": instanceName},
	})

	cysicProofRateGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        "cysic_proof_rate",
		Help:        "1min-proof-rate from cysic log",
		ConstLabels: prometheus.Labels{"instance": instanceName},
	})
}

func monitorOulaLog(logPath, pushgatewayURL, jobName, instanceName, version string, gauge prometheus.Gauge) {
	cmd := exec.Command("tail", "-f", logPath)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(strings.ToLower(line), "total") {
			columns := strings.Fields(line)
			if len(columns) >= 4 {
				value, err := strconv.Atoi(columns[3])
				if err == nil {
					gauge.Set(float64(value))
					pushMetricWithVersion(pushgatewayURL, jobName, version, gauge)
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

func monitorZkworkLog(logPath, pushgatewayURL, jobName, instanceName string) {
	cmd := exec.Command("tail", "-f", logPath)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	re := regexp.MustCompile(`gpu\[\*\]: \(1m - (\d+)`)
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		matches := re.FindStringSubmatch(line)
		if len(matches) > 1 {
			value, err := strconv.Atoi(matches[1])
			if err == nil {
				zkworkGpuGauge.Set(float64(value))
				pushMetric(pushgatewayURL, jobName)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

func monitorCysicLog(logPath, pushgatewayURL, jobName, instanceName string) {
	cmd := exec.Command("tail", "-f", logPath)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	re := regexp.MustCompile(`1min-proof-rate: (\d+)`)
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		matches := re.FindStringSubmatch(line)
		if len(matches) > 1 {
			value, err := strconv.Atoi(matches[1])
			if err == nil {
				cysicProofRateGauge.Set(float64(value))
				pushMetric(pushgatewayURL, jobName)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

func pushMetricWithVersion(pushgatewayURL, jobName, version string, gauge prometheus.Gauge) {
	pusher := push.New(pushgatewayURL, jobName).
		Collector(gauge)

	if err := pusher.Push(); err != nil {
		fmt.Printf("Could not push to Pushgateway: %v\n", err)
	}
}

func pushMetric(pushgatewayURL, jobName string) {
	pusher := push.New(pushgatewayURL, jobName).
		Collector(zkworkGpuGauge).
		Collector(cysicProofRateGauge)

	if err := pusher.Push(); err != nil {
		fmt.Printf("Could not push to Pushgateway: %v\n", err)
	}
}
