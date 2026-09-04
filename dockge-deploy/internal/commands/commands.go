package commands

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

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

func (f *sshFlags) dial() (*sshclient.Client, error) {
	return sshclient.Dial(sshclient.Config{
		Host:             f.host,
		User:             f.user,
		Port:             f.port,
		KeyPath:          f.key,
		Password:         os.Getenv("DOCKGE_DEPLOY_SSH_PASSWORD"),
		KnownHostsPath:   f.knownHosts,
		AcceptNewHostKey: f.acceptNew,
		Timeout:          20 * time.Second,
	})
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
	case "dockge":
		return executeDockge(args[1:])
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func executeWithArgs(fs *flag.FlagSet, flags *sshFlags, args []string, script string, mutating bool) error {
	if err := fs.Parse(args); err != nil {
		return err
	}
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

func executeDockge(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: dockge-deploy dockge <detect|install|upgrade|rollback|plan-migration>")
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
			fmt.Printf("PLAN: install Dockge tag %s into %s, stacks %s, bind %s:%d. Re-run with --apply after review.\n", *tag, *path, *stacks, *bind, *port)
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
	case "rollback":
		fs := flag.NewFlagSet("dockge rollback", flag.ContinueOnError)
		flags := bindSSH(fs)
		path := fs.String("path", "/opt/dockge", "Dockge installation path")
		backup := fs.String("backup", "", "upgrade backup directory to restore")
		apply := fs.Bool("apply", false, "execute rollback")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if *backup == "" {
			return errors.New("--backup is required")
		}
		if !*apply {
			fmt.Printf("PLAN: restore Dockge from backup %s into %s and recreate only Dockge. Re-run with --apply after review.\n", *backup, *path)
			return nil
		}
		return runParsed(flags, remote.RollbackScript(*path, *backup), true)
	case "plan-migration":
		fs := flag.NewFlagSet("dockge plan-migration", flag.ContinueOnError)
		flags := bindSSH(fs)
		source := fs.String("source", "/opt/dockge", "source installation path")
		target := fs.String("target", "/opt/dockge-managed", "proposed target path")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		return runParsed(flags, remote.MigrationPlanScript(*source, *target), false)
	default:
		return fmt.Errorf("unknown dockge command %q", args[0])
	}
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
  dockge detect            find/inspect an existing Dockge installation
  dockge install           plan or install Dockge (requires --apply to mutate)
  dockge upgrade           plan or upgrade Dockge with rollback (requires --apply)
  dockge rollback          restore an upgrade backup (requires --apply)
  dockge plan-migration    read-only migration inventory
  version                  print version

SSH flags:
  --host HOST --user USER --port 22 --key ~/.ssh/id_ed25519
  --known-hosts ~/.ssh/known_hosts [--accept-new-host-key] [--sudo]

Password auth is available only through DOCKGE_DEPLOY_SSH_PASSWORD.
`, appversion.Version)
}
