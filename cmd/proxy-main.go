package cmd

import (
	"context"
	"fmt"
	"github.com/minio/cli"
	"github.com/minio/minio/cmd/logger"
	proxy "github.com/marmotcai/xagent/proxy"
	"os"
	"os/exec"
	. "strings"
)

var proxyFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "address",
		Value: ":" + GlobalXAgentDefaultPort,
		Usage: "bind to a specific ADDRESS:PORT, ADDRESS can be an IP or hostname",
	},
}

var proxyCmd = cli.Command{
	Name:   "proxy",
	Usage:  "start agent proxy",
	Flags:  append(serverFlags, GlobalFlags...),
	Action: proxyMain,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} {{if .VisibleFlags}}[FLAGS] {{end}}DIR1 [DIR2..]
  {{.HelpName}} {{if .VisibleFlags}}[FLAGS] {{end}}DIR{1...64}

DIR:
  DIR points to a directory on a filesystem. When you want to combine
  multiple drives into a single large system, pass one directory per
  filesystem separated by space. You may also use a '...' convention
  to abbreviate the directory arguments. Remote directories in a
  distributed setup are encoded as HTTP(s) URIs.
{{if .VisibleFlags}}
FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}{{end}}
ENVIRONMENT VARIABLES:
  ACCESS:
     XAgent_ACCESS_KEY: Custom username or access key of minimum 3 characters in length.
     XAgent_SECRET_KEY: Custom password or secret key of minimum 8 characters in length.

  BROWSER:
     XAgent_BROWSER: To disable web browser access, set this value to "off".

  CACHE:
     XAgent_CACHE_DRIVES: List of mounted drives or directories delimited by ";".
     XAgent_CACHE_EXCLUDE: List of cache exclusion patterns delimited by ";".
     XAgent_CACHE_EXPIRY: Cache expiry duration in days.
     XAgent_CACHE_MAXUSE: Maximum permitted usage of the cache in percentage (0-100).

  DOMAIN:
     XAgent_DOMAIN: To enable virtual-host-style requests, set this value to XAgent host domain name.

  WORM:
     XAgent_WORM: To turn on Write-Once-Read-Many in server, set this value to "on".

  BUCKET-DNS:
     XAgent_DOMAIN:    To enable bucket DNS requests, set this value to XAgent host domain name.
     XAgent_PUBLIC_IPS: To enable bucket DNS requests, set this value to list of XAgent host public IP(s) delimited by ",".
     XAgent_ETCD_ENDPOINTS: To enable bucket DNS requests, set this value to list of etcd endpoints delimited by ",".

   KMS:
     XAgent_SSE_VAULT_ENDPOINT: To enable Vault as KMS,set this value to Vault endpoint.
     XAgent_SSE_VAULT_APPROLE_ID: To enable Vault as KMS,set this value to Vault AppRole ID.
     XAgent_SSE_VAULT_APPROLE_SECRET: To enable Vault as KMS,set this value to Vault AppRole Secret ID.
     XAgent_SSE_VAULT_KEY_NAME: To enable Vault as KMS,set this value to Vault encryption key-ring name.

EXAMPLES:
  1. Start XAgent server on "/home/shared" directory.
     $ {{.HelpName}} /home/shared

  2. Start XAgent server bound to a specific ADDRESS:PORT.
     $ {{.HelpName}} --address 192.168.1.101:1010 /home/shared

  3. Start XAgent server and enable virtual-host-style requests.
     $ export XAgent_DOMAIN=mydomain.com
     $ {{.HelpName}} --address mydomain.com:1010 /mnt/export

  4. Start erasure coded XAgent server on a node with 64 drives.
     $ {{.HelpName}} /mnt/export{1...64}

  5. Start distributed XAgent server on an 32 node setup with 32 drives each. Run following command on all the 32 nodes.
     $ export XAgent_ACCESS_KEY=XAgent
     $ export XAgent_SECRET_KEY=XAgentstorage
     $ {{.HelpName}} http://node{1...32}.example.com/mnt/export/{1...32}

  6. Start XAgent server with edge caching enabled.
     $ export XAgent_CACHE_DRIVES="/mnt/drive1;/mnt/drive2;/mnt/drive3;/mnt/drive4"
     $ export XAgent_CACHE_EXCLUDE="bucket1/*;*.png"
     $ export XAgent_CACHE_EXPIRY=40
     $ export XAgent_CACHE_MAXUSE=80
     $ {{.HelpName}} /home/shared

  7. Start XAgent server with KMS enabled.
     $ export XAgent_SSE_VAULT_APPROLE_ID=9b56cc08-8258-45d5-24a3-679876769126
     $ export XAgent_SSE_VAULT_APPROLE_SECRET=4e30c52f-13e4-a6f5-0763-d50e8cb4321f
     $ export XAgent_SSE_VAULT_ENDPOINT=https://vault-endpoint-ip:8200
     $ export XAgent_SSE_VAULT_KEY_NAME=my-XAgent-key
     $ {{.HelpName}} /home/shared
`,
}

// This code is from goagain
func lookPath() (argv0 string, err error) {
	argv0, err = exec.LookPath(os.Args[0])
	if nil != err {
		return
	}
	if _, err = os.Stat(argv0); nil != err {
		return
	}
	return
}

func proxyMain(ctx *cli.Context) {
	if ctx.Args().First() == "help" || !endpointsPresent(ctx) {
		cli.ShowCommandHelpAndExit(ctx, "proxy", 1)
	}
	// Handle common command args.
	handleCommonCmdArgs(ctx)

	logger.FatalIf(CheckLocalServerAddr(globalCLIContext.Addr), "Unable to validate passed arguments")

	var setupType SetupType
	var err error

	if len(ctx.Args()) > serverCommandLineArgsMax {
		uErr := uiErrInvalidErasureEndpoints(nil).Msg(fmt.Sprintf("Invalid total number of endpoints (%d) passed, supported upto 32 unique arguments",
			len(ctx.Args())))
		logger.FatalIf(uErr, "Unable to validate passed endpoints")
	}

	endpoints := Fields(os.Getenv("XAGENT_ENDPOINTS"))
	if len(endpoints) > 0 {
		globalXAgentAddr, globalEndpoints, setupType, globalXLSetCount, globalXLSetDriveCount, err = createServerEndpoints(globalCLIContext.Addr, endpoints...)
	} else {
		globalXAgentAddr, globalEndpoints, setupType, globalXLSetCount, globalXLSetDriveCount, err = createServerEndpoints(globalCLIContext.Addr, ctx.Args()...)
	}
	logger.FatalIf(err, "Invalid command line arguments")

	logger.LogIf(context.Background(), checkEndpointsSubOptimal(ctx, setupType, globalEndpoints))

	globalXAgentHost, globalXAgentPort = mustSplitHostPort(globalXAgentAddr)

	proxy.Main()
}
