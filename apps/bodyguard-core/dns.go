package main

import (
	"database/sql"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
)

// DNSServer handles DNS queries for parental control filtering
type DNSServer struct {
	db           *sql.DB
	upstreamDNS  string
	listenAddr   string
	filterEngine *FilterEngine
	urlScanner   *URLScanner
	clamAV       *ClamAVClient
	quitCh       chan struct{}
	running      bool
	mu           sync.RWMutex
}

// NewDNSServer creates a new DNS server
func NewDNSServer(db *sql.DB, filterEngine *FilterEngine) *DNSServer {
	return &DNSServer{
		db:          db,
		upstreamDNS: env("BG_UPSTREAM_DNS", "8.8.8.8:53"),
		listenAddr:  env("BG_DNS_ADDR", ":53"),
		filterEngine: filterEngine,
		urlScanner:  NewURLScanner(),
		clamAV:      NewClamAVClient(),
		quitCh:      make(chan struct{}),
		running:     false,
	}
}

// Start begins listening for DNS queries
func (d *DNSServer) Start() error {
	d.mu.Lock()
	if d.running {
		d.mu.Unlock()
		return fmt.Errorf("DNS server already running")
	}
	d.running = true
	d.mu.Unlock()

	// Create DNS server
	server := &dns.Server{Addr: d.listenAddr, Handler: dns.HandlerFunc(d.handleDNSQuery)}

	log.Printf("[DNS] Starting DNS server on %s (upstream: %s)", d.listenAddr, d.upstreamDNS)
	log.Printf("[DNS] Note: Port 53 requires administrator/root privileges")

	go func() {
		if err := server.ListenAndServe(); err != nil {
			d.mu.Lock()
			d.running = false
			d.mu.Unlock()
			log.Printf("[DNS] Server error: %v", err)
		}
	}()

	return nil
}

// Stop stops the DNS server
func (d *DNSServer) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if !d.running {
		return
	}

	d.running = false
	close(d.quitCh)
	log.Println("[DNS] Stopping DNS server")
}

// IsRunning returns whether the DNS server is running
func (d *DNSServer) IsRunning() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.running
}

// handleDNSQuery processes incoming DNS queries
func (d *DNSServer) handleDNSQuery(w dns.ResponseWriter, r *dns.Msg) {
	msg := new(dns.Msg)
	msg.SetReply(r)

	// Handle all questions in the request
	for _, question := range r.Question {
		qname := strings.TrimSuffix(question.Name, ".")
		qtype := question.Qtype

		log.Printf("[DNS] Query: %s from %s (type: %d)", qname, w.RemoteAddr(), qtype)

		var ruleID string
		var blockReason string

		// Check if domain should be blocked by filter rules
		rule := d.filterEngine.MatchDomain(qname)

		if rule != nil && rule.Enabled {
			// Domain is blocked by filter rule
			blockReason = fmt.Sprintf("Filter: %s", rule.Name)
			ruleID = rule.ID
		} else {
			// Check URL scanner for malicious domains
			scanResult := d.urlScanner.ScanDNSQuery(qname)
			if !scanResult.Allowed {
				blockReason = fmt.Sprintf("Threat Detection: %s", scanResult.Reason)
				log.Printf("[DNS] MALICIOUS DOMAIN DETECTED: %s - %s", qname, scanResult.Reason)
			}
		}

		if blockReason != "" {
			// Block the domain
			log.Printf("[DNS] BLOCKED: %s (reason: %s)", qname, blockReason)

			// Return 0.0.0.0 for A records (NXDOMAIN equivalent for filtering)
			if qtype == dns.TypeA {
				rr := &dns.A{
					Hdr: dns.RR_Header{
						Name:   question.Name,
						Rrtype: qtype,
						Class:  dns.ClassINET,
						Ttl:    300,
					},
					A: net.ParseIP("0.0.0.0"),
				}
				msg.Answer = append(msg.Answer, rr)
			} else if qtype == dns.TypeAAAA {
				// For IPv6, return ::0 (blocked)
				rr := &dns.AAAA{
					Hdr: dns.RR_Header{
						Name:   question.Name,
						Rrtype: qtype,
						Class:  dns.ClassINET,
						Ttl:    300,
					},
					AAAA: net.ParseIP("::0"),
				}
				msg.Answer = append(msg.Answer, rr)
			} else {
				// For other query types, return NXDOMAIN
				msg.SetRcode(r, dns.RcodeNameError)
			}

			// Log the blocked query
			d.logQuery(qname, true, ruleID, blockReason)
			w.WriteMsg(msg)
			return
		}

		// Domain is allowed - forward to upstream DNS
		c := new(dns.Client)
		upstreamQuery := new(dns.Msg)
		upstreamQuery.SetQuestion(question.Name, qtype)
		upstreamQuery.RecursionDesired = true

		response, _, err := c.Exchange(upstreamQuery, d.upstreamDNS)
		if err != nil {
			log.Printf("[DNS] Upstream error: %v", err)
			msg.SetRcode(r, dns.RcodeServerFailure)
			w.WriteMsg(msg)
			return
		}

		// Copy upstream response to our response
		for _, ans := range response.Answer {
			msg.Answer = append(msg.Answer, ans)
		}

		// Log the allowed query
		d.logQuery(qname, false, "", "")
		w.WriteMsg(msg)
	}
}

// logQuery logs DNS queries to the database
func (d *DNSServer) logQuery(domain string, blocked bool, ruleID string, reason string) {
	queryID := fmt.Sprintf("dns-%d", time.Now().UnixNano())

	_, err := d.db.Exec(`
		INSERT INTO dns_queries (id, domain, blocked, rule_id)
		VALUES (?, ?, ?, ?)
	`, queryID, domain, blocked, ruleID)

	if err != nil {
		log.Printf("[DNS] Failed to log query: %v", err)
	}

	// If blocked, log the reason
	if blocked && reason != "" {
		log.Printf("[DNS] Block reason: %s", reason)
	}
}
