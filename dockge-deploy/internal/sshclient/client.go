package sshclient

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

type Config struct {
	Host             string
	User             string
	Port             int
	KeyPath          string
	Password         string
	KnownHostsPath   string
	AcceptNewHostKey bool
	Timeout          time.Duration
}

type Client struct {
	client *ssh.Client
}

func expandHome(path string) string {
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
	}
	return path
}

func hostKeyCallback(path string, acceptNew bool) (ssh.HostKeyCallback, error) {
	path = expandHome(path)
	if path == "" {
		return nil, errors.New("known_hosts path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create known_hosts directory: %w", err)
	}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(path, nil, 0o600); err != nil {
			return nil, fmt.Errorf("create known_hosts: %w", err)
		}
	}
	strict, err := knownhosts.New(path)
	if err != nil {
		return nil, fmt.Errorf("load known_hosts: %w", err)
	}
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		err := strict(hostname, remote, key)
		if err == nil {
			return nil
		}
		var keyErr *knownhosts.KeyError
		if !errors.As(err, &keyErr) {
			return err
		}
		if len(keyErr.Want) > 0 {
			return fmt.Errorf("SSH host key changed for %s; refusing connection", hostname)
		}
		if !acceptNew {
			return fmt.Errorf("unknown SSH host key for %s; verify fingerprint and rerun with --accept-new-host-key: %w", hostname, err)
		}
		line := knownhosts.Line([]string{hostname}, key) + "\n"
		file, openErr := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0o600)
		if openErr != nil {
			return fmt.Errorf("open known_hosts: %w", openErr)
		}
		defer file.Close()
		if _, writeErr := file.WriteString(line); writeErr != nil {
			return fmt.Errorf("append known host: %w", writeErr)
		}
		return nil
	}, nil
}

func authMethods(cfg Config) ([]ssh.AuthMethod, error) {
	var methods []ssh.AuthMethod
	if cfg.KeyPath != "" {
		keyPath := expandHome(cfg.KeyPath)
		if data, err := os.ReadFile(keyPath); err == nil {
			signer, err := ssh.ParsePrivateKey(data)
			if err != nil {
				return nil, fmt.Errorf("parse SSH private key %s: %w", keyPath, err)
			}
			methods = append(methods, ssh.PublicKeys(signer))
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("read SSH private key %s: %w", keyPath, err)
		}
	}
	if cfg.Password != "" {
		methods = append(methods, ssh.Password(cfg.Password))
	}
	if len(methods) == 0 {
		return nil, errors.New("no SSH authentication available; provide --key or DOCKGE_DEPLOY_SSH_PASSWORD")
	}
	return methods, nil
}

func Dial(cfg Config) (*Client, error) {
	if cfg.Host == "" {
		return nil, errors.New("SSH host is required")
	}
	if cfg.User == "" {
		cfg.User = "root"
	}
	if cfg.Port == 0 {
		cfg.Port = 22
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 15 * time.Second
	}
	callback, err := hostKeyCallback(cfg.KnownHostsPath, cfg.AcceptNewHostKey)
	if err != nil {
		return nil, err
	}
	auth, err := authMethods(cfg)
	if err != nil {
		return nil, err
	}
	addr := net.JoinHostPort(cfg.Host, fmt.Sprintf("%d", cfg.Port))
	conn, err := net.DialTimeout("tcp", addr, cfg.Timeout)
	if err != nil {
		return nil, fmt.Errorf("dial SSH %s: %w", addr, err)
	}
	cc, chans, reqs, err := ssh.NewClientConn(conn, addr, &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            auth,
		HostKeyCallback: callback,
		Timeout:         cfg.Timeout,
	})
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("SSH handshake: %w", err)
	}
	return &Client{client: ssh.NewClient(cc, chans, reqs)}, nil
}

func (c *Client) Close() error {
	return c.client.Close()
}

func (c *Client) Run(command string) (string, error) {
	session, err := c.client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()
	output, err := session.CombinedOutput(command)
	if err != nil {
		return string(output), fmt.Errorf("remote command failed: %w", err)
	}
	return string(output), nil
}

func (c *Client) RunScript(script string, sudo bool) (string, error) {
	session, err := c.client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()
	session.Stdin = strings.NewReader(script)
	command := "sh -s"
	if sudo {
		command = "sudo -n sh -s"
	}
	output, err := session.CombinedOutput(command)
	if err != nil {
		return string(output), fmt.Errorf("remote script failed: %w", err)
	}
	return string(output), nil
}
