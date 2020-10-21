package discovery

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path"
	"strconv"
	"time"

	"encoding/binary"
	"github.com/hashicorp/logutils"
)

func initLog(debug bool) {
	log.SetFlags(log.Ldate | log.Ltime)

	if !debug {
		filter := &logutils.LevelFilter{
			Levels:   []logutils.LogLevel{"DEBUG", "INFO", "ERROR", "FATAL"},
			MinLevel: logutils.LogLevel("INFO"),
			Writer:   os.Stderr,
		}
		log.SetOutput(filter)
	}
}

func createDir(dirname string) error {
	fileInfo, err := os.Stat(dirname)
	if err == nil && !fileInfo.IsDir() {
		return fmt.Errorf("%s is aleady existed as file.", dirname)
	} else if fileInfo != nil {
		return nil
	}

	if err := os.Mkdir(dirname, 0777); err != nil {
		return err
	}

	return nil
}

func removeAllFilesUnderDir(dirname string) error {
	files, err := ioutil.ReadDir(dirname)
	if err != nil {
		return err
	}

	for _, file := range files {
		if err := os.Remove(dirname + "/" + file.Name()); err != nil {
			log.Printf("[ERR] can't remove file : %v", err)
		}
	}

	return nil
}

// if need retry, 2nd return value need to set "true"
func hasIPv4Address(ifaceName string) (error, bool) {
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return err, false
	}

	addr := getAddrFromInterface(iface)
	if addr == nil {
		return errors.New("no good IP network found"), true
	} else if addr.IP[0] == 127 {
		return errors.New("skipping localhost"), false
	} else if addr.Mask[0] != 0xff || addr.Mask[1] != 0xff {
		return errors.New("mask means network is too large"), false
	}

	return nil, false
}

func getAddrFromInterface(iface *net.Interface) *net.IPNet {
	var addr *net.IPNet
	if addrs, err := iface.Addrs(); err != nil {
		return nil
	} else {
		for _, a := range addrs {
			if ipnet, ok := a.(*net.IPNet); ok {
				if ip4 := ipnet.IP.To4(); ip4 != nil {
					addr = &net.IPNet{
						IP:   ip4,
						Mask: ipnet.Mask[len(ipnet.Mask)-4:],
					}
					break
				}
			}
		}
	}

	return addr
}

// return all IPv4 addresses within own subnet
func ips(n *net.IPNet) (out []net.IP) {
	num := binary.BigEndian.Uint32([]byte(n.IP))
	mask := binary.BigEndian.Uint32([]byte(n.Mask))
	num &= mask
	for mask < 0xffffffff {
		var buf [4]byte
		binary.BigEndian.PutUint32(buf[:], num)
		out = append(out, net.IP(buf[:]))
		mask++
		num++
	}
	return
}

func scanTCP(ip net.IP, port int) error {
	tcpAddr := ip.String() + ":" + strconv.Itoa(port)
	conn, err := net.DialTimeout("tcp", tcpAddr, 5*time.Second)
	if err != nil {
		return err
	}
	conn.Close()

	return nil
}

func scanUDP(ip net.IP, port int) error {
	addr, err := net.ResolveUDPAddr("udp", ip.String()+":"+strconv.Itoa(port))
	if err != nil {
		return err
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	buf := []byte("PING")
	for i := 0; i < 3; i++ {
		_, err := conn.Write(buf)
		if err != nil {
			return fmt.Errorf("Failed to send UDP4 ping request <%s:%d>", ip.String(), port)
		}
		time.Sleep(time.Second * 1)
	}

	return nil
}

func createAndWriteFile(ip net.IP, port int, hw net.HardwareAddr, dirname string) error {
	filePath := path.Join(dirname, ip.String()+".txt")
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	dst := ip.String() + ":" + strconv.Itoa(port) + ", " + hw.String()
	if _, err := file.WriteString(dst + "\n"); err != nil {
		return err
	}
	log.Printf("[DEBUG] created & writed %s.txt", ip.String())

	return nil
}
