package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	edkeys "github.com/modulrcloud/net-spawner/ed25519"
)

var (
	showHelp = flag.Bool("help", false, "Show help and exit")
	showH    = flag.Bool("h", false, "Show help (shorthand)")
)

type Config struct {
	CorePath string `json:"corePath"` // absolute to the Go node binary
	NetMode  string `json:"netMode"`  // e.g., TESTNET_2V, TESTNET_5V, TESTNET_21V
}

func usage() {
	fmt.Fprintf(os.Stderr, `NetSpawner — local blockchain network launcher

Usage:
  netspawner [flags] <command>

Commands:
  resume   Resume network from the same point
  reset    Reset and start the network from init (progress drop)
  keygen   Generate an Ed25519 key pair as JSON
  help     Show this help

Flags:
  -h, -help   Show help and exit

Examples:
  netspawner resume
  netspawner reset
  netspawner -h
`)
}

func main() {

	flag.Usage = usage

	flag.Parse()

	if *showHelp || *showH {
		usage()
		return
	}
	if flag.NArg() == 0 {
		usage()
		os.Exit(2)
	}

	switch strings.ToLower(flag.Arg(0)) {

	case "help":
		usage()
	case "resume":
		if err := resumeNetwork(); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
	case "reset":
		if err := resetNetwork(); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
	case "keygen":
		if err := runKeygen(flag.Args()[1:]); err != nil {
			if err == flag.ErrHelp {
				return
			}
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %q\n", flag.Arg(0))
		os.Exit(2)

	}
}

func resumeNetwork() error {

	cfg, dir, err := readConfig()

	if err != nil {
		return err
	}

	baseDir := filepath.Join(dir, "X"+cfg.NetMode)

	dirs := CreateDirsForNodes(cfg, baseDir)

	var procs []*exec.Cmd
	for _, nd := range dirs {
		cmd, err := RunCoreProcess(nd, cfg.CorePath)
		if err != nil {
			return fmt.Errorf("spawn for %s: %w", nd, err)
		}
		procs = append(procs, cmd)
	}

	// Block until all children exit.
	for _, p := range procs {
		_ = p.Wait()
	}
	return nil
}

func resetNetwork() error {

	cfg, netSpawnerDir, err := readConfig()

	if err != nil {
		return err
	}

	rootPathForAllNodesChaindata := "X" + cfg.NetMode

	dirForTestnet := filepath.Join(netSpawnerDir, rootPathForAllNodesChaindata)

	srcFilesDir := filepath.Join(netSpawnerDir, "files", "testnets", cfg.NetMode)

	numNodes, err := parseNodesCount(cfg.NetMode)

	if err != nil {
		return err
	}

	if err := ensureDir(dirForTestnet); err != nil {
		return err
	}

	for i := 1; i <= numNodes; i++ {
		nodeDir := filepath.Join(dirForTestnet, "V"+strconv.Itoa(i))
		if err := ensureDir(nodeDir); err != nil {
			return err
		}

		// Copy per-node files into the node root:
		// 1) genesis.json (shared)
		if err := copyFile(
			filepath.Join(srcFilesDir, "genesis.json"),
			filepath.Join(nodeDir, "genesis.json"),
		); err != nil {
			return fmt.Errorf("copy genesis for V%d: %w", i, err)
		}

		// 2) Copy configs_for_nodes/config_i.json to V_i/configs.json

		srcNodeCfg := filepath.Join(srcFilesDir, "configs_for_nodes", fmt.Sprintf("config_%d.json", i))

		if err := copyFile(srcNodeCfg, filepath.Join(nodeDir, "configs.json")); err != nil {
			return fmt.Errorf("copy node config for V%d: %w", i, err)
		}

	}

	fmt.Printf("Directories setup complete for network size %s\n", cfg.NetMode)

	nowMs := time.Now().UnixMilli()

	for i := 1; i <= numNodes; i++ {
		nodeDir := filepath.Join(dirForTestnet, "V"+strconv.Itoa(i))
		genesisPath := filepath.Join(nodeDir, "genesis.json")
		chainDataDir := filepath.Join(nodeDir, "CHAINDATA")

		if fileExists(genesisPath) {
			if err := updateGenesisTimestamp(genesisPath, nowMs); err != nil {
				return fmt.Errorf("update timestamp for %s: %w", genesisPath, err)
			}
			fmt.Printf("Updated timestamp in %s\n", genesisPath)
		}
		if dirExists(chainDataDir) {
			if err := os.RemoveAll(chainDataDir); err != nil {
				return fmt.Errorf("remove CHAINDATA in %s: %w", nodeDir, err)
			}
			fmt.Printf("Deleted CHAINDATA directory in %s\n", nodeDir)
		}
	}

	fmt.Println("Timestamps updated and CHAINDATA directories deleted")

	return resumeNetwork()
}

func runKeygen(args []string) error {
	fs := flag.NewFlagSet("keygen", flag.ContinueOnError)
	mnemonic := fs.String("mnemonic", "", "Existing BIP39 mnemonic. If empty, a new 24-word phrase will be generated")
	passphrase := fs.String("passphrase", "", "Optional mnemonic password")
	derivationPath := fs.String("path", "", "BIP44 derivation path numbers separated by '/' (default 44/7337/0/0)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	var bip44Path []uint32
	if strings.TrimSpace(*derivationPath) != "" {
		parts := strings.Split(*derivationPath, "/")
		bip44Path = make([]uint32, 0, len(parts))
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			val, err := strconv.ParseUint(part, 10, 32)
			if err != nil {
				return fmt.Errorf("invalid BIP44 path component %q: %w", part, err)
			}
			bip44Path = append(bip44Path, uint32(val))
		}
	}

	box := edkeys.GenerateKeyPair(*mnemonic, *passphrase, bip44Path)

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(box)
}
