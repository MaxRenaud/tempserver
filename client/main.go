package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	"github.com/maxrenaud/tempserver/temp"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

type temperature struct {
	location    string
	temperature float32
}

type config struct {
	Ipv4_addr string
	Port      int
	Dst_port  int
}

var cfg *config

func main() {

	debugPtr := flag.Bool("debug", false, "Show debug messages")
	fahrenheitPtr := flag.Bool("fahrenheit", false, "Use Fahrenheit instead of Celsius")
	configPtr := flag.String("config", "client.json", "Path to the config file")
	configOut := flag.String("out", "human","Output format [human, cacti]")

	flag.Parse()
	flag.Lookup("logtostderr").Value.Set("true")
	if *debugPtr {
		flag.Lookup("v").Value.Set("1")
	}
	glog.V(1).Infof("Debug logs: %t\n", *debugPtr)

	glog.V(1).Infoln("Parsing config")
	cfg = &config{}
	parseConfig(configPtr)
	if cfg.Ipv4_addr == "0.0.0.0" {
		err, ipv4 := getRoutableIp()
		checkErr(err)
		cfg.Ipv4_addr = ipv4
	}
	glog.V(1).Infoln("Starting server")
	temps := make(chan temperature, 100)
	real_port := make(chan int, 1)
	go processReplies(temps, real_port)

	glog.V(1).Infoln("Building payload...")
	port := <-real_port
	glog.V(1).Infof("Will listen on %s:%d\n", cfg.Ipv4_addr, port)
	cmd := temp.Command{Command: temp.Command_REQUEST, Address: &temp.Address{Ipv4: cfg.Ipv4_addr, Port: int32(port)}, NodeName: "Client"}
	data, err := proto.Marshal(&cmd)
	if err != nil {
		glog.Exitln("Error marshaling request", err)
	}
	glog.V(1).Infoln("Sending broadcast")

	s, err := net.DialUDP("udp4", nil, &net.UDPAddr{
		IP:   net.IPv4bcast,
		Port: cfg.Dst_port,
	})

	if err != nil {
		glog.Exitln("Error dialing UDP", err)
	}

	_, err = s.Write([]byte(data))
	if err != nil {
		glog.Exitln("Error sending payload", err)
	}

	// Wait 3 second
	time.Sleep(3 * time.Second)
	glog.V(1).Infoln("Done sleeping, let's iterate", len(temps))
	close(temps)
	temp_map := make(map[string]float32)
	for t := range temps {
		if *fahrenheitPtr {
			t.temperature = t.temperature*9/5 + 32
		}
		temp_map[t.location] = t.temperature
		//fmt.Printf("%s: %.2f\n", t.location, t.temperature)
	}

	switch co := *configOut; co {
	case "human":
		for location, temperature := range temp_map {
			fmt.Printf("%s: %.2f\n", location, temperature)
		}
	case "cacti":
		for location, temperature := range temp_map {
			fmt.Printf("%s:%.2f ", location, temperature)
		}
		fmt.Println()

	}

}

func getRoutableIp() (error, string) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return err, ""
	}
	for _, addr := range addrs {
		if ip, ok := addr.(*net.IPNet); ok && !ip.IP.IsLoopback() {
			if ip.IP.To4() != nil {
				return nil, ip.IP.String()
			}
		}

	}
	return errors.New("Couldn't get non-loopback IPv4"), ""
}

func checkErr(err error) {
	if err != nil {
		glog.Fatal(err)
	}
}

func checkExit(err error) {
	if err != nil {
		glog.Exit(err)
	}
}

func parseConfig(location *string) {
	file, err := os.Open(*location)
	checkExit(err)

	decoder := json.NewDecoder(file)
	err = decoder.Decode(cfg)
	checkExit(err)
}

func processReplies(temps chan temperature, real_port chan int) {
	saddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", cfg.Ipv4_addr, cfg.Port))
	if err != nil {
		glog.Fatal(err)
	}
	sbind, err := net.ListenUDP("udp", saddr)
	if err != nil {
		glog.Fatal(err)
	}
	port, err := strconv.Atoi(strings.Split(sbind.LocalAddr().String(), ":")[1])
	checkErr(err)
	real_port <- port
	defer sbind.Close()
	buf := make([]byte, 1024)

	for {
		n, src, err := sbind.ReadFromUDP(buf)
		glog.V(1).Infof("ReceivedFrom %s\n", src)
		if err != nil {
			glog.Fatal(err)
		}
		command := temp.Command{}
		if err := proto.Unmarshal(buf[:n], &command); err != nil {
			glog.Errorln("Unmarshal error", err)
		}
		if command.Command == temp.Command_REPLY {
			temps <- temperature{
				location:    command.NodeName,
				temperature: command.Temperature.Temperature,
			}
		}
	}

}
