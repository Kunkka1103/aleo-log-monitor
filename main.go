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

func main() {
	// 定义命令行参数
	oulaLogPath := flag.String("oula-log", "oula.log", "Path to the oula log file")
	zkworkLogPath := flag.String("zkwork-log", "zkwork.log", "Path to the zkwork log file")
	cysicLogPath := flag.String("cysic-log", "cysic.log", "Path to the cysic log file")
	pushgatewayURL := flag.String("pushgateway-url", "http://localhost:9091", "Pushgateway URL")
	jobName := flag.String("job-name", "log_monitor", "Job name for Pushgateway")
	instanceName := flag.String("instance-name", "instance1", "Instance name for Pushgateway")

	flag.Parse()

	// 启动日志监控
	go monitorOulaLog(*oulaLogPath, *pushgatewayURL, *jobName, *instanceName)
	go monitorZkworkLog(*zkworkLogPath, *pushgatewayURL, *jobName, *instanceName)
	go monitorCysicLog(*cysicLogPath, *pushgatewayURL, *jobName, *instanceName)

	// 保持程序运行
	select {}
}

func monitorOulaLog(logPath, pushgatewayURL, jobName, instanceName string) {
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
					pushMetric("oula_total", float64(value), pushgatewayURL, jobName, instanceName)
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
				pushMetric("zkwork_gpu", float64(value), pushgatewayURL, jobName, instanceName)
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
				pushMetric("cysic_proof_rate", float64(value), pushgatewayURL, jobName, instanceName)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

func pushMetric(metricName string, value float64, pushgatewayURL, jobName, instanceName string) {
	pusher := push.New(pushgatewayURL, jobName).
		Collector(prometheus.NewGauge(prometheus.GaugeOpts{
			Name: metricName,
			ConstLabels: prometheus.Labels{
				"instance": instanceName,
			},
		}))

	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: metricName,
	})
	gauge.Set(value)
	pusher.Collector(gauge)

	if err := pusher.Push(); err != nil {
		fmt.Printf("Could not push to Pushgateway: %v\n", err)
	}
}
