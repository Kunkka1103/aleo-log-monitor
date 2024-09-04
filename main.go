package main

import (
	"bufio"
	"flag"
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
	oulaLogPath := flag.String("oula-log", "", "Path to the oula log file")
	oulaNewLogPath := flag.String("oula-new-log", "", "Path to the new version oula log file")
	zkworkLogPath := flag.String("zkwork-log", "", "Path to the zkwork log file")
	cysicLogPath := flag.String("cysic-log", "", "Path to the cysic log file")
	koiLogPath := flag.String("koi-log", "", "Path to the koi log file")
	zkpoolLogPath := flag.String("zkpool-log", "", "Path to the zkpool log file")
	pushgatewayURL := flag.String("pushgateway-url", "http://localhost:9091", "Pushgateway URL")
	instance := flag.String("instance", "", "Instance name")

	flag.Parse()

	// 启动日志监控
	if *oulaLogPath != "" {
		log.Printf("Starting monitoring for Oula log: %s", *oulaLogPath)
		go monitorOulaLog(*oulaLogPath, "oula_total_v1", *instance, *pushgatewayURL)
	} else {
		log.Println("Oula log path not provided, skipping...")
	}

	if *oulaNewLogPath != "" {
		log.Printf("Starting monitoring for new version Oula log: %s", *oulaNewLogPath)
		go monitorOulaLog(*oulaNewLogPath, "oula_total_v2", *instance, *pushgatewayURL)
	} else {
		log.Println("New version Oula log path not provided, skipping...")
	}

	if *zkworkLogPath != "" {
		log.Printf("Starting monitoring for Zkwork log: %s", *zkworkLogPath)
		go monitorZkworkLog(*zkworkLogPath, "zkwork_gpu", *instance, *pushgatewayURL)
	} else {
		log.Println("Zkwork log path not provided, skipping...")
	}

	if *cysicLogPath != "" {
		log.Printf("Starting monitoring for Cysic log: %s", *cysicLogPath)
		go monitorCysicLog(*cysicLogPath, "cysic_proof_rate", *instance, *pushgatewayURL)
	} else {
		log.Println("Cysic log path not provided, skipping...")
	}

	if *koiLogPath != "" {
		log.Printf("Starting monitoring for Koi log: %s", *koiLogPath)
		go monitorKoiLog(*koiLogPath, "koi_instant_rate", *instance, *pushgatewayURL)
	} else {
		log.Println("Koi log path not provided, skipping...")
	}

	if *zkpoolLogPath != "" {
		log.Printf("Starting monitoring for Zkpool log: %s", *zkpoolLogPath)
		go monitorZkpoolLog(*zkpoolLogPath, "f2pool_proof_rate", *instance, *pushgatewayURL)
	} else {
		log.Println("Zkpool log path not provided, skipping...")
	}

	// 保持程序运行
	select {}
}

func monitorOulaLog(logPath string, jobName string, instance string, url string) {
	cmd := exec.Command("tail", "-f", logPath)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("Failed to start tail command for %s: %v", logPath, err)
	}
	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start monitoring for %s: %v", logPath, err)
	}

	log.Printf("Monitoring Oula log: %s", logPath)
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		columns := strings.Fields(line)
		if len(columns) >= 4 && strings.Contains(strings.ToLower(columns[0]), "total") {
			value, err := strconv.Atoi(columns[3])
			if err == nil {
				log.Printf("Extracted value from Oula log (%s): %d", logPath, value)
				Push(jobName, instance, float64(value), url)
			} else {
				log.Printf("Failed to convert value from Oula log (%s): %v", logPath, err)
			}
		} else {
			log.Printf("Unexpected log format in Oula log (%s): %s", logPath, line)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading Oula log (%s): %v", logPath, err)
	}
}

func monitorZkworkLog(logPath string, jobName string, instance string, url string) {
	cmd := exec.Command("tail", "-f", logPath)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("Failed to start tail command for %s: %v", logPath, err)
	}
	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start monitoring for %s: %v", logPath, err)
	}

	log.Printf("Monitoring Zkwork log: %s", logPath)
	re := regexp.MustCompile(`gpu\[\*\]: \(1m - (\d+)`)
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		matches := re.FindStringSubmatch(line)
		if len(matches) > 1 {
			value, err := strconv.Atoi(matches[1])
			if err == nil {
				log.Printf("Extracted GPU value from Zkwork log (%s): %d", logPath, value)
				Push(jobName, instance, float64(value), url)
			} else {
				log.Printf("Failed to convert GPU value from Zkwork log (%s): %v", logPath, err)
			}
		} else {
			log.Printf("Unexpected log format in Zkwork log (%s): %s", logPath, line)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading Zkwork log (%s): %v", logPath, err)
	}
}

func monitorCysicLog(logPath string, jobName string, instance string, url string) {
	cmd := exec.Command("tail", "-f", logPath)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("Failed to start tail command for %s: %v", logPath, err)
	}
	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start monitoring for %s: %v", logPath, err)
	}

	log.Printf("Monitoring Cysic log: %s", logPath)
	re := regexp.MustCompile(`1min-proof-rate: (\d+)`)
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		matches := re.FindStringSubmatch(line)
		if len(matches) > 1 {
			value, err := strconv.Atoi(matches[1])
			if err == nil {
				log.Printf("Extracted 1min-proof-rate from Cysic log (%s): %d", logPath, value)
				Push(jobName, instance, float64(value), url)
			} else {
				log.Printf("Failed to convert 1min-proof-rate from Cysic log (%s): %v", logPath, err)
			}
		} else {
			log.Printf("Unexpected log format in Cysic log (%s): %s", logPath, line)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading Cysic log (%s): %v", logPath, err)
	}
}

func monitorKoiLog(logPath string, jobName string, instance string, url string) {
	cmd := exec.Command("tail", "-f", logPath)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("Failed to start tail command for %s: %v", logPath, err)
	}
	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start monitoring for %s: %v", logPath, err)
	}

	log.Printf("Monitoring Koi log: %s", logPath)
	re := regexp.MustCompile(`instant rate: ([\d\.]+)`)
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		matches := re.FindStringSubmatch(line)
		if len(matches) > 1 {
			value, err := strconv.ParseFloat(matches[1], 64)
			if err == nil {
				log.Printf("Extracted instant rate from Koi log (%s): %f", logPath, value)
				Push(jobName, instance, value, url)
			} else {
				log.Printf("Failed to convert instant rate from Koi log (%s): %v", logPath, err)
			}
		} else {
			log.Printf("Unexpected log format in Koi log (%s): %s", logPath, line)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading Koi log (%s): %v", logPath, err)
	}
}

func monitorZkpoolLog(logPath string, jobName string, instance string, url string) {
	cmd := exec.Command("tail", "-f", logPath)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("Failed to start tail command for %s: %v", logPath, err)
	}
	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start monitoring for %s: %v", logPath, err)
	}

	log.Printf("Monitoring Zkpool log: %s", logPath)
	re := regexp.MustCompile(`proof rate (\d+)/s`)
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		matches := re.FindStringSubmatch(line)
		if len(matches) > 1 {
			value, err := strconv.Atoi(matches[1])
			if err == nil {
				log.Printf("Extracted proof rate from Zkpool log (%s): %d", logPath, value)
				Push(jobName, instance, float64(value), url)
			} else {
				log.Printf("Failed to convert proof rate from Zkpool log (%s): %v", logPath, err)
			}
		} else {
			log.Printf("Unexpected log format in Zkpool log (%s): %s", logPath, line)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading Zkpool log (%s): %v", logPath, err)
	}
}

func Push(jobName string, instance string, value float64, url string) {
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{Name: jobName})
	gauge.Set(value)
	err := push.New(url, jobName).
		Grouping("instance", instance).
		Collector(gauge).
		Push()
	if err != nil {
		log.Printf("Push to Prometheus %s failed: %s", url, err)
	} else {
		log.Printf("Successfully pushed metric to %s: %s=%f", url, jobName, value)
	}
}
