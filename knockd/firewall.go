package main

import (
	"fmt"
	"log"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Firewall is an interface for managing firewall rules.
type Firewall interface {
	// Add temporarily adds a rule to the firewall for a given IP and ports.
	// The rule should be automatically deleted after the ttl (in minutes) expires.
	Add(ip string, ports []int, ttl int) error
	// Del explicitly removes a firewall rule.
	Del(ip string, ports []int) error
	// Cleanup removes all rules managed by this firewall instance.
	Cleanup() error
}

// newFirewall creates a new firewall instance based on the operating system.
func newFirewall(goos string) Firewall {
	switch goos {
	case "linux":
		log.Println("Using iptables firewall for Linux")
		return &linuxFirewall{}
	case "windows":
		log.Println("Using netsh advfirewall for Windows")
		return &windowsFirewall{}
	default:
		log.Fatalf("Unsupported OS for firewall management: %s", goos)
		return nil
	}
}

// --- Linux Firewall (iptables) ---

type linuxFirewall struct {
	mu        sync.Mutex
	activeIPs map[string]bool
}

func (f *linuxFirewall) Add(ip string, ports []int, ttl int) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	log.Printf("[FIREWALL] Adding rule for IP: %s, Ports: %v, TTL: %d minutes", ip, ports, ttl)
	if err := f.runIPTables(true, ip, ports); err != nil {
		return err
	}

	// Track active IP
	if f.activeIPs == nil {
		f.activeIPs = make(map[string]bool)
	}
	f.activeIPs[ip] = true

	// Schedule the deletion of the rule
	go func() {
		time.Sleep(time.Duration(ttl) * time.Minute)
		log.Printf("[FIREWALL] TTL expired. Deleting rule for IP: %s, Ports: %v", ip, ports)
		if err := f.Del(ip, ports); err != nil {
			log.Printf("[FIREWALL] Error deleting expired rule for %s: %v", ip, err)
		}
	}()

	return nil
}

func (f *linuxFirewall) Del(ip string, ports []int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	// Remove from active IPs
	if f.activeIPs != nil {
		delete(f.activeIPs, ip)
	}
	
	return f.runIPTables(false, ip, ports)
}

// Cleanup removes all active firewall rules created by this instance
func (f *linuxFirewall) Cleanup() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	log.Println("[FIREWALL] Cleaning up knockd-managed firewall rules...")
	
	// Only clean rules we actually created and are tracking
	for ip := range f.activeIPs {
		// Use iptables with comment to identify our rules
		cmd := exec.Command("iptables", "-L", "INPUT", "--line-numbers")
		output, err := cmd.Output()
		if err != nil {
			log.Printf("[FIREWALL] Error listing rules: %v", err)
			continue
		}
		
		// Parse output to find our rules and remove them safely
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, "knockd-allow") && strings.Contains(line, ip) {
				// Extract line number and remove safely
				fields := strings.Fields(line)
				if len(fields) > 0 {
					lineNum := fields[0]
					if lineNum != "num" && lineNum != "Chain" { // Skip header
						cmd := exec.Command("iptables", "-D", "INPUT", lineNum)
						if err := cmd.Run(); err != nil {
							log.Printf("[FIREWALL] Error removing rule %s for %s: %v", lineNum, ip, err)
						}
					}
				}
			}
		}
	}
	
	f.activeIPs = make(map[string]bool)
	return nil
}

func (f *linuxFirewall) runIPTables(add bool, ip string, ports []int) error {
	// Validate IP address format
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return fmt.Errorf("invalid IP address format: %s", ip)
	}
	
	// Validate and format ports
	var validPorts []string
	for _, port := range ports {
		if port < 1 || port > 65535 {
			return fmt.Errorf("invalid port number: %d", port)
		}
		validPorts = append(validPorts, strconv.Itoa(port))
	}
	portsStr := strings.Join(validPorts, ",")
	
	// Use -I to insert at the top, -D to delete
	operation := "-D"
	if add {
		operation = "-I"
	}

	cmd := exec.Command("iptables", operation, "INPUT", "-s", parsedIP.String(), "-p", "tcp", "-m", "multiport", "--dports", portsStr, "-m", "comment", "--comment", "knockd-allow", "-j", "ACCEPT")
	log.Printf("[FIREWALL] Executing: %s", cmd.String())

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("iptables command failed: %s, output: %s", err, string(output))
	}
	return nil
}

// --- Windows Firewall (netsh) ---

type windowsFirewall struct {
	mu        sync.Mutex
	activeIPs map[string]bool
}

func (f *windowsFirewall) getRuleName(ip string) string {
	return fmt.Sprintf("knockd-allow-%s", ip)
}

func (f *windowsFirewall) Add(ip string, ports []int, ttl int) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	log.Printf("[FIREWALL] Adding rule for IP: %s, Ports: %v, TTL: %d minutes", ip, ports, ttl)
	
	// Validate IP address format
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return fmt.Errorf("invalid IP address format: %s", ip)
	}
	
	// Validate and format ports
	var validPorts []string
	for _, port := range ports {
		if port < 1 || port > 65535 {
			return fmt.Errorf("invalid port number: %d", port)
		}
		validPorts = append(validPorts, strconv.Itoa(port))
	}
	portsStr := strings.Join(validPorts, ",")
	
	// Track active IP
	if f.activeIPs == nil {
		f.activeIPs = make(map[string]bool)
	}
	f.activeIPs[parsedIP.String()] = true

	// On Windows, we first delete any pre-existing rule for this IP to ensure a clean state.
	_ = f.Del(ip, ports)

	ruleName := f.getRuleName(parsedIP.String())

	cmd := exec.Command("netsh", "advfirewall", "firewall", "add", "rule",
		fmt.Sprintf("name=%s", ruleName),
		"dir=in",
		"action=allow",
		"protocol=TCP",
		fmt.Sprintf("remoteip=%s", parsedIP.String()),
		fmt.Sprintf("localport=%s", portsStr),
	)
	log.Printf("[FIREWALL] Executing: %s", cmd.String())

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("netsh add command failed: %s, output: %s", err, string(output))
	}

	// Schedule the deletion
	go func() {
		time.Sleep(time.Duration(ttl) * time.Minute)
		log.Printf("[FIREWALL] TTL expired. Deleting rule for IP: %s, Ports: %v", ip, ports)
		if err := f.Del(ip, ports); err != nil {
			log.Printf("[FIREWALL] Error deleting expired rule for %s: %v", ip, err)
		}
	}()

	return nil
}

func (f *windowsFirewall) Del(ip string, ports []int) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Validate IP address format
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return fmt.Errorf("invalid IP address format: %s", ip)
	}
	
	// Remove from active IPs
	if f.activeIPs != nil {
		delete(f.activeIPs, parsedIP.String())
	}
	
	ruleName := f.getRuleName(parsedIP.String())
	cmd := exec.Command("netsh", "advfirewall", "firewall", "delete", "rule", fmt.Sprintf("name=%s", ruleName))
	log.Printf("[FIREWALL] Executing: %s", cmd.String())

	output, err := cmd.CombinedOutput()
	if err != nil {
		// It's common for this to fail if the rule doesn't exist, so we don't return a hard error.
		log.Printf("[FIREWALL] Note: 'netsh delete' command finished with (possible) error: %s, output: %s", err, string(output))
	}
	return nil
}

// Cleanup removes all active firewall rules created by this instance
func (f *windowsFirewall) Cleanup() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	log.Println("[FIREWALL] Cleaning up knockd-managed firewall rules...")
	for ip := range f.activeIPs {
		ruleName := f.getRuleName(ip)
		cmd := exec.Command("netsh", "advfirewall", "firewall", "delete", "rule", fmt.Sprintf("name=%s", ruleName))
		log.Printf("[FIREWALL] Executing: %s", cmd.String())

		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("[FIREWALL] Error cleaning up rule for %s: %s, output: %s", ip, err, string(output))
		}
	}
	f.activeIPs = make(map[string]bool)
	return nil
}