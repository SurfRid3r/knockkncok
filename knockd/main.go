package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/google/gopacket/layers"
)

func deriveKeys(masterKey []byte) ([]byte, []byte) {
	hmacE := hmac.New(sha256.New, masterKey)
	hmacE.Write([]byte("knockknock-encrypt"))
	keyE := hmacE.Sum(nil)

	hmacH := hmac.New(sha256.New, masterKey)
	hmacH.Write([]byte("knockknock-hmac"))
	keyH := hmacH.Sum(nil)

	return keyE, keyH
}

func main() {
	cfg, err := LoadConfig("knockd.toml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	var masterKey []byte
	if cfg.Key == "" {
		log.Println("Master key not found in config, generating a new one...")
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			log.Fatalf("Failed to generate new key: %v", err)
		}
		cfg.Key = base64.StdEncoding.EncodeToString(key)
		log.Println("Please add the following line to your knockd.toml file:")
		log.Printf(`key = "%s"`, cfg.Key)
	}
	masterKey, err = base64.StdEncoding.DecodeString(cfg.Key)
	if err != nil {
		log.Fatalf("Failed to decode master key: %v", err)
	}
	if len(masterKey) != 32 {
		log.Fatalf("Invalid master key length: expected 32 bytes, got %d", len(masterKey))
	}

	if cfg.Iface == "" {
		log.Println("Interface not specified, attempting to auto-select...")
		cfg.Iface, err = autoSelectInterface()
		if err != nil {
			log.Fatalf("Failed to auto-select interface: %v. Please specify it in knockd.toml", err)
		}
		log.Printf("Automatically selected interface: %s", cfg.Iface)
	}

	keyE, keyH := deriveKeys(masterKey)

	fw := newFirewall(runtime.GOOS)
	defer func() {
		log.Println("[MAIN] Cleaning up firewall rules...")
		if err := fw.Cleanup(); err != nil {
			log.Printf("[MAIN] Error during cleanup: %v", err)
		}
	}()

	sn, err := NewSniffer(cfg.Iface)
	if err != nil {
		log.Fatalf("Failed to create sniffer: %v", err)
	}
	defer sn.Close()

	db, err := NewDB(cfg.DbFile)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("Failed to close database: %v", err)
		}
	}()

	ttlEngine := NewTTLEngine(cfg.BaseTTLMin, cfg.MaxTTLMin, db)

	nonceStore := NewNonceStore(time.Minute)

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	log.Println("knockd is running...")

	// Main packet processing loop
	for {
		select {
		case pkt, ok := <-sn.C():
			if !ok {
				log.Println("Sniffer channel closed, shutting down...")
				return
			}

			ipLayer := pkt.Layer(layers.LayerTypeIPv4)
			if ipLayer == nil {
				continue
			}
			ip, ok := ipLayer.(*layers.IPv4)
			if !ok {
				log.Printf("Failed to assert IPv4 layer from packet")
				continue
			}

			info, ok := Verify(pkt.Data(), keyE, keyH, nonceStore)
			if !ok {
				continue
			}
			info.IP = ip.SrcIP.String()

			ttl := ttlEngine.Next(info.AgentID, info.IP)
			if err := fw.Add(info.IP, cfg.AllowPorts, ttl); err != nil {
				log.Printf("Failed to add firewall rule for %s: %v", info.IP, err)
			} else {
				log.Printf("Added firewall rule for %s with TTL %d minutes", info.IP, ttl)
			}
			if err := db.IncrementScore(info.AgentID, info.IP); err != nil {
				log.Printf("Failed to increment score for agent %d, IP %s: %v", info.AgentID, info.IP, err)
			}

		case sig := <-sigChan:
			log.Printf("Received signal %v, shutting down gracefully...", sig)
			return
		}
	}
}
