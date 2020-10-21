package discovery

import (
	"log"
	"net"
	"time"

	"github.com/avast/retry-go"
	"github.com/kelseyhightower/envconfig"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
)

type ServiceDiscovery struct {
	IfaceName  string
	Port       int
	Dirname    string
	ScanType   string
	RetryCnt   uint
	Debug      bool
	OutputType string
	Env        EnvironmentValues
}

type Addr struct {
	IP           net.IP
	HardwareAddr net.HardwareAddr
}

type EnvironmentValues struct {
	MysqlUser     string `envconfig:"MYSQL_USER" default:"latona"`
	MysqlPassword string `envconfig:"MYSQL_PASSWORD"`
	MysqlHost     string `envconfig:"MYSQL_HOST" default:"127.0.0.1"`
	MysqlPort     int    `envconfig:"MYSQL_PORD" default:"30000"`
	MysqlDB       string `envconfig:"MYSQ_DB" default:"Device"`
	MysqlTable    string `envconfig:"MYSQL_TABLE" default:"device"`
}

const (
	packageName       = "distributed-service-discovery"
	defaultIfaceName  = "eth0"
	defaultPort       = 10039
	defaultDirname    = "/var/local/distributed-service-discovery"
	defaultScanType   = "tcp"
	defaultDebug      = false
	defaultRetryCount = 15
	defaultOutputType = "file"
	checkRetryDelay   = 30
	statusAlive       = 0
	statusDead        = 1
)

// NewServiceDiscoveryCommand creates a *cobra.Command object with default parameters
func NewServiceDiscoveryCommand() *cobra.Command {
	serviceDiscovery := newServiceDiscovery()

	cmd := &cobra.Command{
		Use:   packageName,
		Short: "run distributed service discovery",
		Run: func(cmd *cobra.Command, args []string) {
			serviceDiscovery.run()
		},
	}

	fs := cmd.Flags()
	serviceDiscovery.set(fs)

	return cmd
}

func newServiceDiscovery() *ServiceDiscovery {
	return &ServiceDiscovery{
		IfaceName:  defaultIfaceName,
		Port:       defaultPort,
		Dirname:    defaultDirname,
		Debug:      defaultDebug,
		RetryCnt:   defaultRetryCount,
		ScanType:   defaultScanType,
		OutputType: defaultOutputType,
	}
}

func (s *ServiceDiscovery) set(fs *flag.FlagSet) {
	fs.StringVarP(&s.IfaceName, "interface", "i", s.IfaceName, "interface name")
	fs.IntVarP(&s.Port, "port", "p", s.Port, "target port")
	fs.StringVar(&s.Dirname, "dirname", s.Dirname, "directory to output file writed IPv4 List")
	fs.UintVarP(&s.RetryCnt, "retrycnt", "r", s.RetryCnt, "retry count when interface has no IPv4 Address")
	fs.StringVarP(&s.ScanType, "scantype", "s", s.ScanType, "scan type(tcp or udp)")
	fs.StringVarP(&s.OutputType, "output", "o", s.OutputType, "output type(file or mysql)")

	fs.BoolVarP(&s.Debug, "debug", "d", s.Debug, "debug mode")
}

func (s *ServiceDiscovery) run() {
	initLog(s.Debug)

	if err := envconfig.Process("", &s.Env); err != nil {
		log.Fatalf("cant load environment variable: %v", err)
	}

	if err := removeAllFilesUnderDir(s.Dirname); err != nil {
		log.Fatalf("[FATAL] %v", err)
	}
	if err := createDir(s.Dirname); err != nil {
		log.Fatalf("[FATAL] %v", err)
	}

	err, isRetry := hasIPv4Address(s.IfaceName)
	if err != nil && !isRetry {
		log.Fatalf("[FATAL] %v", err)
	} else if err != nil && isRetry {
		if err := retry.Do(
			func() error {
				if err, _ := hasIPv4Address(s.IfaceName); err != nil {
					return err
				}
				return nil
			},
			retry.DelayType(func(n uint, config *retry.Config) time.Duration {
				log.Print("[INFO] retry to check IPv4 Address...")
				return time.Duration(checkRetryDelay) * time.Second
			}),
			retry.Attempts(s.RetryCnt),
		); err != nil {
			log.Fatalf("[FATAL] %v", err)
		}
	}

	iface, err := net.InterfaceByName(s.IfaceName)
	if err != nil {
		log.Fatalf("[FATAL] %v", err)
	}
	addrCh := make(chan *Addr)

	go s.arp(iface, getAddrFromInterface(iface), addrCh)

	var conn *MyDB
	if s.OutputType == "mysql" {
		conn, err = NewConnection(s.Env.MysqlUser, s.Env.MysqlPassword, s.Env.MysqlHost, s.Env.MysqlPort, s.Env.MysqlDB)
		if err != nil {
			log.Fatalf("[FATAL] %v", err)
		}
		defer conn.CloseConnection()

		_, err := conn.db.Query("DELETE FROM " + s.Env.MysqlTable)
		if err != nil {
			log.Fatalf("[FATAL] %v", err)
		}
	}

	// loop until finding IPs running gossip-propagation-d or check all ip within own subnet
	stopCounter := 0
loop:
	for {
		select {
		case addr := <-addrCh:
			stopCounter = 0
			if s.ScanType == "tcp" {
				if err := scanTCP(addr.IP, s.Port); err != nil {
					log.Printf("[INFO] %v", err)
					continue
				}
			} else if s.ScanType == "udp" {
				if err := scanUDP(addr.IP, s.Port); err != nil {
					log.Printf("[INFO] %v", err)
					continue
				}
			}

			if s.OutputType == defaultOutputType {
				if err := createAndWriteFile(addr.IP, s.Port, addr.HardwareAddr, s.Dirname); err != nil {
					log.Printf("[WARN] %v", err)
					continue
				}
			} else if s.OutputType == "mysql" {
				in, err := conn.db.Prepare("INSERT INTO " + s.Env.MysqlTable + "(macAddress, deviceIp, connectionStatus) VALUES(?,?,?)")
				if err != nil {
					log.Printf("[WARN] %v", err)
					continue
				}

				row, err := in.Exec(addr.HardwareAddr.String(), addr.IP.String(), statusAlive)
				if err != nil {
					log.Printf("[WARN] %v", err)
					continue
				}

				log.Printf("[INFO] success to insert %v", row)
			}

		default:
			if stopCounter > 2 {
				break loop
			}
			stopCounter++
			time.Sleep(1 * time.Second)
		}
	}
}
