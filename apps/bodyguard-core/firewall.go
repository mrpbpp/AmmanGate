package main

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// FirewallAction defines the interface for firewall operations
type FirewallAction interface {
	Block(ip string) error
	Unblock(ip string) error
}

// WindowsFirewall implements firewall actions for Windows
type WindowsFirewall struct{}

// Block creates a firewall block rule for the given IP
func (w *WindowsFirewall) Block(ip string) error {
	// Remove any existing rule first
	_ = w.Unblock(ip)

	// Create new block rule
	ruleName := fmt.Sprintf("AmmanGate-Block-%s", strings.ReplaceAll(ip, ".", "-"))
	args := []string{
		"advfirewall",
		"firewall",
		"add",
		"rule",
		fmt.Sprintf(`name="%s"`, ruleName),
		"dir=out",
		"action=block",
		fmt.Sprintf("remoteip=%s", ip),
	}

	cmd := exec.Command("netsh", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("netsh failed: %w, output: %s", err, string(output))
	}

	// Also block inbound traffic
	argsIn := []string{
		"advfirewall",
		"firewall",
		"add",
		"rule",
		fmt.Sprintf(`name="%s-In"`, ruleName),
		"dir=in",
		"action=block",
		fmt.Sprintf("remoteip=%s", ip),
	}

	cmdIn := exec.Command("netsh", argsIn...)
	outputIn, err := cmdIn.CombinedOutput()
	if err != nil {
		return fmt.Errorf("netsh inbound failed: %w, output: %s", err, string(outputIn))
	}

	return nil
}

// Unblock removes the firewall block rule for the given IP
func (w *WindowsFirewall) Unblock(ip string) error {
	ruleName := fmt.Sprintf("AmmanGate-Block-%s", strings.ReplaceAll(ip, ".", "-"))

	// Remove outbound rule
	argsOut := []string{
		"advfirewall",
		"firewall",
		"delete",
		"rule",
		fmt.Sprintf(`name="%s"`, ruleName),
	}

	cmdOut := exec.Command("netsh", argsOut...)
	_ = cmdOut.Run() // Ignore error if rule doesn't exist

	// Remove inbound rule
	argsIn := []string{
		"advfirewall",
		"firewall",
		"delete",
		"rule",
		fmt.Sprintf(`name="%s-In"`, ruleName),
	}

	cmdIn := exec.Command("netsh", argsIn...)
	_ = cmdIn.Run() // Ignore error if rule doesn't exist

	return nil
}

// LinuxFirewall implements firewall actions for Linux using iptables
type LinuxFirewall struct{}

// Block creates an iptables rule to block the IP
func (l *LinuxFirewall) Block(ip string) error {
	// Check if rule already exists
	checkCmd := exec.Command("iptables", "-C", "INPUT", "-s", ip, "-j", "DROP")
	_ = checkCmd.Run()

	// Add DROP rule for INPUT
	argsIn := []string{"-I", "INPUT", "-s", ip, "-j", "DROP"}
	cmdIn := exec.Command("iptables", argsIn...)
	if output, err := cmdIn.CombinedOutput(); err != nil {
		return fmt.Errorf("iptables INPUT failed: %w, output: %s", err, string(output))
	}

	// Add DROP rule for OUTPUT
	argsOut := []string{"-I", "OUTPUT", "-d", ip, "-j", "DROP"}
	cmdOut := exec.Command("iptables", argsOut...)
	if output, err := cmdOut.CombinedOutput(); err != nil {
		return fmt.Errorf("iptables OUTPUT failed: %w, output: %s", err, string(output))
	}

	return nil
}

// Unblock removes the iptables rule for the IP
func (l *LinuxFirewall) Unblock(ip string) error {
	// Remove INPUT rule
	argsIn := []string{"-D", "INPUT", "-s", ip, "-j", "DROP"}
	cmdIn := exec.Command("iptables", argsIn...)
	_ = cmdIn.Run() // Ignore error if rule doesn't exist

	// Remove OUTPUT rule
	argsOut := []string{"-D", "OUTPUT", "-d", ip, "-j", "DROP"}
	cmdOut := exec.Command("iptables", argsOut...)
	_ = cmdOut.Run() // Ignore error if rule doesn't exist

	return nil
}

// GetFirewall returns the appropriate firewall implementation for the OS
func GetFirewall() FirewallAction {
	switch runtime.GOOS {
	case "windows":
		return &WindowsFirewall{}
	case "linux":
		return &LinuxFirewall{}
	default:
		// For unsupported OS, return a no-op firewall
		return &NoOpFirewall{}
	}
}

// NoOpFirewall is a no-op implementation for unsupported platforms
type NoOpFirewall struct{}

func (n *NoOpFirewall) Block(ip string) error {
	return fmt.Errorf("firewall operations not supported on %s", runtime.GOOS)
}

func (n *NoOpFirewall) Unblock(ip string) error {
	return fmt.Errorf("firewall operations not supported on %s", runtime.GOOS)
}

// executeFirewallBlock is a helper function that blocks an IP using the system firewall
func executeFirewallBlock(target string) error {
	fw := GetFirewall()
	return fw.Block(target)
}

// executeFirewallUnblock is a helper function that unblocks an IP using the system firewall
func executeFirewallUnblock(target string) error {
	fw := GetFirewall()
	return fw.Unblock(target)
}
