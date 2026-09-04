package sshclient

import (
	"crypto/ed25519"
	"crypto/rand"
	"net"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
)

func testPublicKey(t *testing.T) ssh.PublicKey {
	t.Helper()
	public, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	key, err := ssh.NewPublicKey(public)
	if err != nil {
		t.Fatalf("convert key: %v", err)
	}
	return key
}

func TestHostKeyTrustOnFirstUseAndChangeDetection(t *testing.T) {
	knownHosts := t.TempDir() + "/known_hosts"
	hostname := "example.test:22"
	remote := &net.TCPAddr{IP: net.ParseIP("203.0.113.10"), Port: 22}
	firstKey := testPublicKey(t)
	firstFingerprint := ssh.FingerprintSHA256(firstKey)

	strict, err := hostKeyCallback(knownHosts, false)
	if err != nil {
		t.Fatalf("create strict callback: %v", err)
	}
	err = strict(hostname, remote, firstKey)
	if err == nil {
		t.Fatal("unknown host key should be rejected without explicit acceptance")
	}
	if !strings.Contains(err.Error(), firstFingerprint) {
		t.Fatalf("unknown-host error must expose fingerprint %q: %v", firstFingerprint, err)
	}

	accept, err := hostKeyCallback(knownHosts, true)
	if err != nil {
		t.Fatalf("create accepting callback: %v", err)
	}
	if err := accept(hostname, remote, firstKey); err != nil {
		t.Fatalf("explicit first-host acceptance failed: %v", err)
	}

	strict, err = hostKeyCallback(knownHosts, false)
	if err != nil {
		t.Fatalf("reload strict callback: %v", err)
	}
	if err := strict(hostname, remote, firstKey); err != nil {
		t.Fatalf("persisted host key should be accepted: %v", err)
	}

	changedKey := testPublicKey(t)
	changedFingerprint := ssh.FingerprintSHA256(changedKey)
	err = strict(hostname, remote, changedKey)
	if err == nil {
		t.Fatal("changed host key must be rejected")
	}
	if !strings.Contains(err.Error(), "changed") || !strings.Contains(err.Error(), changedFingerprint) {
		t.Fatalf("changed-key error must identify change and received fingerprint: %v", err)
	}
}
