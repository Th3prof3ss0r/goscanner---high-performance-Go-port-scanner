// Command goscanner is a high-performance network port scanner.
//
// Usage:
//
//	goscanner [flags] <target> [target ...]
//
// Examples:
//
//	goscanner 192.168.1.1
//	goscanner -p 1-1024 -sV 192.168.1.0/24
//	goscanner -p- --rate 50000 -w 2000 target.com
//	sudo goscanner -sS -p 1-10000 192.168.1.1
//	goscanner -iL hosts.txt --top-ports 1000 -f json -o report.json
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/user/goscanner/output"
	"github.com/user/goscanner/scanner"
	"github.com/user/goscanner/utils"
)

// version is the goscanner version string.
const version = "1.0.0"

func main() {
	os.Exit(run())
}

func run() int {
	// ── Flag definitions ───────────────────────────────────────────────────
	var (
		portSpec  = flag.String("p", "", "Port range: single (80), range (1-1024), list (80,443), all (-)")
		topPorts  = flag.Int("top-ports", 0, "Scan top N most common ports (overrides -p)")
		timeout   = flag.Int("t", 500, "Connection timeout in milliseconds")
		workers   = flag.Int("w", 1000, "Number of concurrent workers")
		rate      = flag.Int("rate", 10000, "Maximum packets per second (token bucket)")
		outputFile = flag.String("o", "", "Output file path (default: stdout)")
		format    = flag.String("f", "table", "Output format: table | json | xml | csv")
		inputFile = flag.String("iL", "", "Read targets from file, one per line")
		synScan   = flag.Bool("sS", false, "SYN scan — stealth (requires root/sudo)")
		udpScan   = flag.Bool("sU", false, "UDP scan in addition to TCP")
		sVersion  = flag.Bool("sV", false, "Service/version detection via banner grabbing")
		osDetect  = flag.Bool("O", false, "OS fingerprint detection")
		skipPing  = flag.Bool("Pn", false, "Skip host discovery (scan even if host appears down)")
		verbose   = flag.Bool("v", false, "Verbose: include closed/filtered ports in output")
		openOnly  = flag.Bool("open", false, "Show only open ports")
		ipv6      = flag.Bool("6", false, "Enable IPv6 scanning")
		ver       = flag.Bool("version", false, "Print version and exit")
	)

	// Custom usage
	flag.Usage = usage
	flag.Parse()

	if *ver {
		fmt.Printf("goscanner %s\n", version)
		return 0
	}

	// ── Logging ────────────────────────────────────────────────────────────
	logLevel := slog.LevelInfo
	if *verbose {
		logLevel = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel})))

	// ── Collect targets ────────────────────────────────────────────────────
	var targetArgs []string

	if *inputFile != "" {
		ips, err := utils.ParseTargetFile(*inputFile)
		if err != nil {
			slog.Error("read target file", "err", err)
			return 1
		}
		targetArgs = append(targetArgs, ips...)
	}

	// Remaining positional arguments
	for _, arg := range flag.Args() {
		targetArgs = append(targetArgs, arg)
	}

	if len(targetArgs) == 0 {
		fmt.Fprintln(os.Stderr, "error: no targets specified")
		flag.Usage()
		return 1
	}

	// Resolve all targets to individual IP addresses
	targets, err := utils.ParseTargets(targetArgs)
	if err != nil {
		slog.Error("parse targets", "err", err)
		return 1
	}
	if len(targets) == 0 {
		fmt.Fprintln(os.Stderr, "error: targets resolved to 0 hosts")
		return 1
	}

	// ── Resolve ports ──────────────────────────────────────────────────────
	var ports []int

	switch {
	case *topPorts > 0:
		ports = utils.TopPorts(*topPorts)
	case *portSpec != "":
		ports, err = utils.ParsePorts(*portSpec)
		if err != nil {
			slog.Error("parse ports", "err", err)
			return 1
		}
	default:
		ports = utils.TopPorts(1000)
	}

	// ── Print scan banner ──────────────────────────────────────────────────
	scanType := "TCP Connect"
	if *synScan {
		scanType = "SYN Stealth"
	} else if *udpScan {
		scanType = "TCP+UDP"
	}
	fmt.Printf("\nGoScanner %s — %s scan\n", version, scanType)
	fmt.Printf("Targets: %d host(s) | Ports: %d | Workers: %d | Rate: %d pps | Timeout: %dms\n\n",
		len(targets), len(ports), *workers, *rate, *timeout)

	// ── Build configuration ────────────────────────────────────────────────
	cfg := &scanner.Config{
		Ports:         ports,
		Timeout:       *timeout,
		Workers:       clamp(*workers, 1, 65535),
		Rate:          clamp(*rate, 1, 1000000),
		SynScan:       *synScan,
		UDPScan:       *udpScan,
		VersionDetect: *sVersion,
		OSDetect:      *osDetect,
		SkipPing:      *skipPing,
		OpenOnly:      *openOnly,
		Verbose:       *verbose,
		IPv6:          *ipv6,
		Format:        *format,
		Output:        *outputFile,
	}

	// ── Run scan ───────────────────────────────────────────────────────────
	s := scanner.NewScanner(cfg)
	reports, err := s.Scan(targets)
	if err != nil {
		slog.Error("scan failed", "err", err)
		return 1
	}

	// ── Output results ─────────────────────────────────────────────────────
	rCfg := output.ReportConfig{
		Format: strings.ToLower(*format),
		Path:   *outputFile,
	}
	if err := output.WriteReport(reports, rCfg); err != nil {
		slog.Error("write report", "err", err)
		return 1
	}

	return 0
}

func usage() {
	fmt.Fprintf(os.Stderr, `GoScanner %s — High-performance port scanner

Usage:
  goscanner [flags] <target> [target ...]

Targets:
  Single IP:   192.168.1.1
  CIDR block:  192.168.1.0/24
  IP range:    192.168.1.1-254
  Hostname:    example.com
  From file:   -iL hosts.txt

Port Specifications:
  -p 80             Single port
  -p 1-1024         Port range
  -p 80,443,8080    Port list
  -p-               All 65535 ports
  --top-ports N     Top N most common ports (default: 1000)

Scan Types:
  (default)    TCP Connect scan — full 3-way handshake
  -sS          SYN scan (stealth, requires root/sudo)
  -sU          Also scan UDP ports

Detection:
  -sV          Service/version detection (banner grabbing)
  -O           OS fingerprint detection

Performance:
  -w N         Concurrent workers (default: 1000)
  -t N         Timeout in ms (default: 500)
  --rate N     Max packets per second (default: 10000)

Output:
  -f FORMAT    table | json | xml | csv (default: table)
  -o FILE      Write output to file
  --open       Show only open ports
  -v           Verbose (include closed/filtered)
  -Pn          Skip ping/host discovery

Other:
  -iL FILE     Read targets from file
  -6           Enable IPv6
  --version    Show version

Examples:
  goscanner 192.168.1.1
  goscanner -p 1-1024 -sV 192.168.1.0/24
  sudo goscanner -sS -p- --rate 100000 10.0.0.1
  goscanner --top-ports 100 -f json -o out.json -iL targets.txt
  goscanner -p 80,443,8080-8090 -sV -O example.com

`, version)
	os.Exit(2)
}

// clamp returns v clamped to [lo, hi].
func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
