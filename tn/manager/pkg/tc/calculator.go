package tc

import (
	"fmt"
	"math"
)

// Rule represents a TC (Traffic Control) rule configuration
type Rule struct {
	Interface   string  `json:"interface"`
	Direction   string  `json:"direction"` // ingress or egress
	RateKbit    int     `json:"rateKbit"`
	BurstKB     int     `json:"burstKB"`
	LatencyMs   float32 `json:"latencyMs"`
	JitterMs    float32 `json:"jitterMs,omitempty"`
	LossPercent float32 `json:"lossPercent,omitempty"`
	Priority    int32   `json:"priority"`
	Handle      string  `json:"handle"`
	Parent      string  `json:"parent"`
	Classid     string  `json:"classid"`
	TCCommands  []string `json:"tcCommands"` // Actual tc commands to execute
}

// Calculator calculates TC parameters based on QoS requirements
type Calculator struct {
	// OverheadFactor accounts for VXLAN and other protocol overhead
	OverheadFactor float32
}

// NewCalculator creates a new TC parameter calculator
func NewCalculator() *Calculator {
	return &Calculator{
		OverheadFactor: 1.1, // 10% overhead for VXLAN encapsulation
	}
}

// CalculateRules generates TC rules based on QoS parameters
func (c *Calculator) CalculateRules(bandwidth, latency, jitter, loss float32, priority int32) []Rule {
	var rules []Rule

	// Calculate adjusted bandwidth accounting for overhead
	adjustedBandwidth := bandwidth * c.OverheadFactor
	rateKbit := int(adjustedBandwidth * 1000)

	// Calculate burst size (important for small packets)
	// Burst = rate * latency / 8 (approximately)
	burstKB := c.calculateBurst(rateKbit, latency)

	// Generate egress rules (main shaping happens here)
	egressRule := c.generateEgressRule(rateKbit, burstKB, latency, jitter, loss, priority)
	rules = append(rules, egressRule)

	// Generate ingress rules (policing)
	ingressRule := c.generateIngressRule(rateKbit, burstKB, priority)
	rules = append(rules, ingressRule)

	return rules
}

func (c *Calculator) generateEgressRule(rateKbit, burstKB int, latency, jitter, loss float32, priority int32) Rule {
	rule := Rule{
		Interface:   "vxlan0", // Will be replaced with actual interface
		Direction:   "egress",
		RateKbit:    rateKbit,
		BurstKB:     burstKB,
		LatencyMs:   latency,
		JitterMs:    jitter,
		LossPercent: loss,
		Priority:    priority,
		Handle:      "1:",
		Parent:      "root",
		Classid:     fmt.Sprintf("1:%d", priority),
	}

	// Generate TC commands
	rule.TCCommands = c.generateTCCommands(rule)
	return rule
}

func (c *Calculator) generateIngressRule(rateKbit, burstKB int, priority int32) Rule {
	rule := Rule{
		Interface: "vxlan0",
		Direction: "ingress",
		RateKbit:  rateKbit,
		BurstKB:   burstKB,
		Priority:  priority,
		Handle:    "ffff:",
		Parent:    "ingress",
	}

	// Ingress policing commands
	rule.TCCommands = []string{
		fmt.Sprintf("tc qdisc add dev %s handle ffff: ingress", rule.Interface),
		fmt.Sprintf("tc filter add dev %s parent ffff: protocol ip prio %d u32 match ip src 0.0.0.0/0 "+
			"police rate %dkbit burst %dk drop flowid :1",
			rule.Interface, priority, rateKbit, burstKB),
	}

	return rule
}

func (c *Calculator) generateTCCommands(rule Rule) []string {
	var commands []string
	iface := rule.Interface

	if rule.Direction == "egress" {
		// Root qdisc - HTB for hierarchical shaping
		commands = append(commands,
			fmt.Sprintf("tc qdisc add dev %s root handle %s htb default 30", iface, rule.Handle))

		// Root class
		commands = append(commands,
			fmt.Sprintf("tc class add dev %s parent %s classid 1:1 htb rate %dkbit ceil %dkbit",
				iface, rule.Handle, rateKbit*10, rateKbit*10)) // Total bandwidth

		// Slice-specific class with rate limiting
		commands = append(commands,
			fmt.Sprintf("tc class add dev %s parent 1:1 classid %s htb rate %dkbit ceil %dkbit burst %dk prio %d",
				iface, rule.Classid, rule.RateKbit, rule.RateKbit, rule.BurstKB, rule.Priority))

		// Add netem for latency, jitter, and loss
		netemCmd := fmt.Sprintf("tc qdisc add dev %s parent %s handle %d: netem",
			iface, rule.Classid, int(rule.Priority)*10)

		// Add delay
		netemCmd += fmt.Sprintf(" delay %.1fms", rule.LatencyMs)

		// Add jitter if specified
		if rule.JitterMs > 0 {
			netemCmd += fmt.Sprintf(" %.1fms 25%%", rule.JitterMs) // 25% correlation
		}

		// Add packet loss if specified
		if rule.LossPercent > 0 {
			netemCmd += fmt.Sprintf(" loss %.2f%%", rule.LossPercent)
		}

		commands = append(commands, netemCmd)

		// Add filter to classify packets
		commands = append(commands,
			fmt.Sprintf("tc filter add dev %s protocol ip parent %s prio %d u32 match ip dst 0.0.0.0/0 flowid %s",
				iface, rule.Handle, rule.Priority, rule.Classid))
	}

	return commands
}

func (c *Calculator) calculateBurst(rateKbit int, latencyMs float32) int {
	// Burst size calculation:
	// burst = rate * RTT / 8
	// We use a fraction of latency as RTT estimate
	rttEstimate := latencyMs / 1000.0 // Convert to seconds
	burstBytes := float64(rateKbit) * 1000 / 8 * float64(rttEstimate)

	// Minimum burst size (at least 1 MTU worth)
	minBurst := 1500.0

	// Add some headroom
	burst := math.Max(burstBytes*1.5, minBurst)

	return int(burst / 1024) // Convert to KB
}

// GenerateProfileCommands generates TC commands for predefined profiles
func (c *Calculator) GenerateProfileCommands(profile string, iface string) []string {
	var commands []string

	switch profile {
	case "eMBB":
		// Profile 1: 4.57 Mbps, 16.1ms latency
		commands = []string{
			fmt.Sprintf("tc qdisc add dev %s root handle 1: htb default 30", iface),
			fmt.Sprintf("tc class add dev %s parent 1: classid 1:1 htb rate 10000kbit", iface),
			fmt.Sprintf("tc class add dev %s parent 1:1 classid 1:10 htb rate 4570kbit ceil 4570kbit burst 15k", iface),
			fmt.Sprintf("tc qdisc add dev %s parent 1:10 handle 10: netem delay 16.1ms 2ms", iface),
			fmt.Sprintf("tc filter add dev %s protocol ip parent 1: prio 1 u32 match ip dst 0.0.0.0/0 flowid 1:10", iface),
		}

	case "mIoT":
		// Profile 2: 2.77 Mbps, 15.7ms latency
		commands = []string{
			fmt.Sprintf("tc qdisc add dev %s root handle 1: htb default 30", iface),
			fmt.Sprintf("tc class add dev %s parent 1: classid 1:1 htb rate 10000kbit", iface),
			fmt.Sprintf("tc class add dev %s parent 1:1 classid 1:20 htb rate 2770kbit ceil 2770kbit burst 10k", iface),
			fmt.Sprintf("tc qdisc add dev %s parent 1:20 handle 20: netem delay 15.7ms 2ms", iface),
			fmt.Sprintf("tc filter add dev %s protocol ip parent 1: prio 2 u32 match ip dst 0.0.0.0/0 flowid 1:20", iface),
		}

	case "uRLLC":
		// Profile 3: 0.93 Mbps, 6.3ms latency
		commands = []string{
			fmt.Sprintf("tc qdisc add dev %s root handle 1: htb default 30", iface),
			fmt.Sprintf("tc class add dev %s parent 1: classid 1:1 htb rate 10000kbit", iface),
			fmt.Sprintf("tc class add dev %s parent 1:1 classid 1:30 htb rate 930kbit ceil 930kbit burst 5k", iface),
			fmt.Sprintf("tc qdisc add dev %s parent 1:30 handle 30: netem delay 6.3ms 1ms", iface),
			fmt.Sprintf("tc filter add dev %s protocol ip parent 1: prio 3 u32 match ip dst 0.0.0.0/0 flowid 1:30", iface),
		}

	default:
		// Default profile
		commands = []string{
			fmt.Sprintf("tc qdisc add dev %s root handle 1: htb default 30", iface),
		}
	}

	return commands
}

// CleanupCommands generates commands to remove TC configuration
func (c *Calculator) CleanupCommands(iface string) []string {
	return []string{
		fmt.Sprintf("tc qdisc del dev %s root 2>/dev/null || true", iface),
		fmt.Sprintf("tc qdisc del dev %s ingress 2>/dev/null || true", iface),
	}
}