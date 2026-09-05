package commands

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/wkarts/dockge/dockge-deploy/internal/dockgeapi"
	"github.com/wkarts/dockge/dockge-deploy/internal/remote"
	"github.com/wkarts/dockge/dockge-deploy/internal/sshclient"
	appversion "github.com/wkarts/dockge/dockge-deploy/internal/version"
)

type sshFlags struct {
	host       string
	user       string
	port       int
	key        string
	knownHosts string
	acceptNew  bool
	sudo       bool
}

type apiFlags struct {
	url         string
	allowHTTP   bool
	insecureTLS bool
}

func bindSSH(fs *flag.FlagSet) *sshFlags {
	f := &sshFlags{}
	fs.StringVar(&f.host, "host", "", "SSH host or IP")
	fs.StringVar(&f.user, "user", "root", "SSH user")
	fs.IntVar(&f.port, "port", 22, "SSH port")
	fs.StringVar(&f.key, "key", "~/.ssh/id_ed25519", "SSH private key path")
	fs.StringVar(&f.knownHosts, "known-hosts", "~/.ssh/known_hosts", "known_hosts path")
	fs.BoolVar(&f.acceptNew, "accept-new-host-key", false, "trust and persist a previously unknown host key; changed keys are always rejected")
	fs.BoolVar(&f.sudo, "sudo", false, "run mutating remote scripts through sudo -n")
	return f
}

func bindAPI(fs *flag.FlagSet) *apiFlags {
	f := &apiFlags{}
	fs.StringVar(&f.url, "url", "", "Dockge base URL, for example https://dockge.example.com")
	fs.BoolVar(&f.allowHTTP, "allow-http", false, "allow plain HTTP for a trusted private network; loopback is always allowed")
	fs.BoolVar(&f.insecureTLS, "insecure-tls", false, "disable TLS certificate verification; lab use only")
	return f
}

func (f *sshFlags) dial() (*sshclient.Client, error) {
	return sshclient.Dial(sshclient.Config{
		Host:             f.host,
		User:             f.user,
		Port:             f.port,
		KeyPath:          f.key,
		KeyPassphrase:    os.Getenv("DOCKGE_DEPLOY_SSH_KEY_PASSPHRASE"),
		Password:         os.Getenv("DOCKGE_DEPLOY_SSH_PASSWORD"),
		KnownHostsPath:   f.knownHosts,
		AcceptNewHostKey: f.acceptNew,
		Timeout:          20 * time.Second,
	})
}

func (f *apiFlags) client() (*dockgeapi.Client, error) {
	if strings.TrimSpace(f.url) == "" {
		return nil, errors.New("--url is required")
	}
	return dockgeapi.New(
		f.url,
		os.Getenv("DOCKGE_DEPLOY_DOCKGE_TOKEN"),
		f.allowHTTP,
		f.insecureTLS,
	)
}

func Execute(args []string) error {
	if len(args) == 0 {
		usage()
		return nil
	}
	switch args[0] {
	case "version", "--version", "-v":
		fmt.Println(appversion.Version)
		return nil
	case "host":
		if len(args) < 2 || args[1] != "inspect" {
			return errors.New("usage: dockge-deploy host inspect [SSH flags]")
		}
		fs := flag.NewFlagSet("host inspect", flag.ContinueOnError)
		flags := bindSSH(fs)
		return executeWithArgs(fs, flags, args[2:], remote.InspectScript(), false)
	case "doctor":
		fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
		flags := bindSSH(fs)
		return executeWithArgs(fs, flags, args[1:], remote.InspectScript(), false)
	case "docker":
		return executeDocker(args[1:])
	case "dockge":
		return executeDockge(args[1:])
	case "stack":
		return executeStack(args[1:])
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func executeWithArgs(fs *flag.FlagSet, flags *sshFlags, args []string, script string, mutating bool) error {
	if err := fs.Parse(args); err != nil {
		return err
	}
	return runParsed(flags, script, mutating)
}

func executeDocker(args []string) error {
	if len(args) == 0 || args[0] != "install" {
		return errors.New("usage: dockge-deploy docker install [SSH flags] [--apply]")
	}
	fs := flag.NewFlagSet("docker install", flag.ContinueOnError)
	flags := bindSSH(fs)
	apply := fs.Bool("apply", false, "install Docker Engine and Compose when absent")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if !*apply {
		fmt.Println("PLAN: detect distribution/package manager, install Docker Engine + Compose v2 only when missing, enable Docker service, then verify docker and docker compose. Re-run with --apply after review.")
		return nil
	}
	return runParsed(flags, remote.DockerInstallScript(), true)
}

func executeDockge(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: dockge-deploy dockge <detect|install|upgrade|migrate|rollback|plan-migration|manager-token>")
	}
	switch args[0] {
	case "detect":
		fs := flag.NewFlagSet("dockge detect", flag.ContinueOnError)
		flags := bindSSH(fs)
		path := fs.String("path", "/opt/dockge", "Dockge installation path")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		return runParsed(flags, remote.DetectScript(*path), false)
	case "install":
		fs := flag.NewFlagSet("dockge install", flag.ContinueOnError)
		flags := bindSSH(fs)
		path := fs.String("path", "/opt/dockge", "Dockge installation path")
		stacks := fs.String("stacks-path", "/opt/stacks", "Dockge stacks path")
		tag := fs.String("version", "latest", "Dockge image tag")
		bind := fs.String("bind-host", "127.0.0.1", "Dockge bind host")
		port := fs.Int("dockge-port", 5001, "Dockge port")
		apply := fs.Bool("apply", false, "execute the installation")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if !*apply {
			fmt.Printf("PLAN: install Dockge tag %s into %s, stacks %s, bind %s:%d. Existing installations are never overwritten. Re-run with --apply after review.\n", *tag, *path, *stacks, *bind, *port)
			return nil
		}
		return runParsed(flags, remote.InstallScript(*path, *stacks, *tag, *bind, *port), true)
	case "upgrade":
		fs := flag.NewFlagSet("dockge upgrade", flag.ContinueOnError)
		flags := bindSSH(fs)
		path := fs.String("path", "/opt/dockge", "Dockge installation path")
		tag := fs.String("version", "latest", "target Dockge image tag")
		apply := fs.Bool("apply", false, "execute upgrade with snapshot and rollback")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if !*apply {
			fmt.Println(remote.UpgradePlan(*path, *tag))
			return nil
		}
		return runParsed(flags, remote.UpgradeScript(*path, *tag), true)
	case "migrate":
		fs := flag.NewFlagSet("dockge migrate", flag.ContinueOnError)
		flags := bindSSH(fs)
		path := fs.String("path", "/opt/dockge", "existing Dockge installation path to migrate in-place")
		stacks := fs.String("stacks-path", "/opt/stacks", "existing stacks path to preserve")
		tag := fs.String("version", "latest", "wkarts/dockge target tag")
		bind := fs.String("bind-host", "127.0.0.1", "new Dockge bind host")
		port := fs.Int("dockge-port", 5001, "new Dockge port")
		apply := fs.Bool("apply", false, "execute the reviewed in-place migration")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if !*apply {
			fmt.Println(remote.MigrationPlan(*path, *stacks, *tag, *bind, *port))
			return nil
		}
		return runParsed(flags, remote.MigrationScript(*path, *stacks, *tag, *bind, *port), true)
	case "rollback":
		fs := flag.NewFlagSet("dockge rollback", flag.ContinueOnError)
		flags := bindSSH(fs)
		path := fs.String("path", "/opt/dockge", "Dockge installation path")
		backup := fs.String("backup", "", "upgrade/migration backup directory to restore")
		apply := fs.Bool("apply", false, "execute rollback")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if *backup == "" {
			return errors.New("--backup is required")
		}
		if !*apply {
			fmt.Printf("PLAN: restore Dockge from backup %s into %s and recreate only Dockge. Managed application stacks/volumes are not removed. Re-run with --apply after review.\n", *backup, *path)
			return nil
		}
		return runParsed(flags, remote.RollbackScript(*path, *backup), true)
	case "plan-migration":
		fs := flag.NewFlagSet("dockge plan-migration", flag.ContinueOnError)
		flags := bindSSH(fs)
		path := fs.String("path", "/opt/dockge", "existing installation path")
		stacks := fs.String("stacks-path", "/opt/stacks", "stacks path")
		tag := fs.String("version", "latest", "wkarts/dockge target tag")
		bind := fs.String("bind-host", "127.0.0.1", "proposed bind host")
		port := fs.Int("dockge-port", 5001, "proposed Dockge port")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		return runParsed(flags, remote.MigrationPlanScript(*path, *stacks, *tag, *bind, *port), false)
	case "manager-token":
		fs := flag.NewFlagSet("dockge manager-token", flag.ContinueOnError)
		flags := bindSSH(fs)
		path := fs.String("path", "/opt/dockge", "Dockge installation path")
		name := fs.String("name", "dockge-manager", "Automation API credential name")
		prefixes := fs.String("prefixes", allStackPrefixes(), "comma-separated stack prefixes; default covers all valid stack initial characters")
		replace := fs.Bool("replace", false, "rotate an existing active credential with the same name")
		apply := fs.Bool("apply", false, "create/rotate the credential and print its one-time secret")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if !*apply {
			fmt.Printf("PLAN: create Dockge Automation API credential %q in %s with Manager scopes and %d namespace prefixes. Secret is printed once. Re-run with --apply after review.\n", *name, *path, len(strings.Split(*prefixes, ",")))
			return nil
		}
		return runParsed(flags, remote.ManagerTokenScript(*path, *name, *prefixes, *replace), true)
	default:
		return fmt.Errorf("unknown dockge command %q", args[0])
	}
}

func executeStack(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: dockge-deploy stack <list|inspect|logs|apply|start|stop|restart|pull|up|down>")
	}
	cmd := args[0]
	fs := flag.NewFlagSet("stack "+cmd, flag.ContinueOnError)
	flags := bindAPI(fs)
	name := fs.String("name", "", "stack name")
	tail := fs.Int("tail", 200, "log tail lines (1-2000)")
	composeFile := fs.String("compose", "", "local compose.yaml path")
	envFile := fs.String("env", "", "optional local .env path")
	adopt := fs.Bool("adopt", false, "explicitly adopt an existing external stack when applying")
	apply := fs.Bool("apply", false, "execute a mutating action")
	idem := fs.String("idempotency-key", "", "stable key for retrying exactly the same mutation; generated when omitted")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	client, err := flags.client()
	if err != nil {
		return err
	}

	var payload []byte
	switch cmd {
	case "list":
		payload, err = client.Stacks()
	case "inspect":
		if *name == "" {
			return errors.New("--name is required")
		}
		payload, err = client.Stack(*name)
	case "logs":
		if *name == "" {
			return errors.New("--name is required")
		}
		if *tail < 1 || *tail > 2000 {
			return errors.New("--tail must be between 1 and 2000")
		}
		payload, err = client.Logs(*name, *tail)
	case "apply":
		if *name == "" || *composeFile == "" {
			return errors.New("--name and --compose are required")
		}
		if !*apply {
			fmt.Printf("PLAN: apply local Compose %s to stack %s through Dockge Automation API; adopt=%t. Re-run with --apply after review.\n", *composeFile, *name, *adopt)
			return nil
		}
		composeYAML, readErr := os.ReadFile(*composeFile)
		if readErr != nil {
			return fmt.Errorf("read compose file: %w", readErr)
		}
		var composeEnv *string
		if *envFile != "" {
			envBytes, readErr := os.ReadFile(*envFile)
			if readErr != nil {
				return fmt.Errorf("read env file: %w", readErr)
			}
			value := string(envBytes)
			composeEnv = &value
		}
		key, keyErr := mutationKey(*idem)
		if keyErr != nil {
			return keyErr
		}
		payload, err = client.Apply(*name, string(composeYAML), composeEnv, *adopt, key)
	case "start", "stop", "restart", "pull", "up", "down":
		if *name == "" {
			return errors.New("--name is required")
		}
		if !*apply {
			fmt.Printf("PLAN: execute stack action %s on %s through Dockge Automation API. Re-run with --apply after review.\n", cmd, *name)
			return nil
		}
		key, keyErr := mutationKey(*idem)
		if keyErr != nil {
			return keyErr
		}
		payload, err = client.Action(*name, cmd, key)
	default:
		return fmt.Errorf("unknown stack command %q", cmd)
	}
	if err != nil {
		return err
	}
	printJSON(payload)
	return nil
}

func mutationKey(value string) (string, error) {
	if strings.TrimSpace(value) != "" {
		return value, nil
	}
	return dockgeapi.NewIdempotencyKey()
}

func printJSON(payload []byte) {
	var value any
	if json.Unmarshal(payload, &value) == nil {
		if formatted, err := json.MarshalIndent(value, "", "  "); err == nil {
			fmt.Println(string(formatted))
			return
		}
	}
	fmt.Println(string(payload))
}

func allStackPrefixes() string {
	return "0,1,2,3,4,5,6,7,8,9,a,b,c,d,e,f,g,h,i,j,k,l,m,n,o,p,q,r,s,t,u,v,w,x,y,z"
}

func runParsed(flags *sshFlags, script string, mutating bool) error {
	if flags.host == "" {
		return errors.New("--host is required")
	}
	client, err := flags.dial()
	if err != nil {
		return err
	}
	defer client.Close()
	output, err := client.RunScript(script, mutating && flags.sudo)
	if output != "" {
		fmt.Print(output)
	}
	return err
}

func usage() {
	fmt.Printf(`Dockge Deploy %s

Commands:
  host inspect             inspect Linux host and Docker
  doctor                   inspect host prerequisites
  docker install           install Docker Engine + Compose when absent (plan-first)
  dockge detect            find/inspect an existing Dockge installation
  dockge install           install Dockge (requires --apply to mutate)
  dockge upgrade           upgrade wkarts/dockge with snapshot/rollback
  dockge plan-migration    read-only legacy migration inventory
  dockge migrate           migrate an existing Dockge in-place to wkarts/dockge
  dockge rollback          restore an upgrade/migration backup
  dockge manager-token     create a one-time Automation API credential for Manager
  stack list               list stacks through Dockge Automation API
  stack inspect            inspect one stack through the API
  stack logs               read stack logs through the API
  stack apply              deploy/update Compose through the API (plan-first)
  stack start|stop|restart|pull|up|down  operate a stack (plan-first)
  version                  print version

SSH flags:
  --host HOST --user USER --port 22 --key ~/.ssh/id_ed25519
  --known-hosts ~/.ssh/known_hosts [--accept-new-host-key] [--sudo]

API flags:
  --url https://dockge.example.com [--allow-http] [--insecure-tls]
  Bearer token is read only from DOCKGE_DEPLOY_DOCKGE_TOKEN.

SSH_AUTH_SOCK is used automatically when an OpenSSH-compatible agent is available.
Password auth is read only from DOCKGE_DEPLOY_SSH_PASSWORD.
Encrypted key passphrases are read only from DOCKGE_DEPLOY_SSH_KEY_PASSPHRASE.
`, appversion.Version)
}
