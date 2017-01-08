package main

import (
	"fmt"
	"net"
	"github.com/golang/glog"
	"github.com/maxrenaud/tempserver/temp"
	"github.com/tarm/serial"
	"github.com/yryz/ds18b20"
	"errors"
	"os"
	"encoding/json"
	"github.com/golang/protobuf/proto"
	"time"
	"strconv"
	"flag"
)

type Sensor interface {
	getTemp() (error, float32)
}

type tempAlert struct {
	USB string
}

type ds18b20_sensor struct {
}

type fake_temp struct {
	desired float32
}

type config struct {
	Name          string
	Ipv4_addr     string
	Port          int
	Sensor        string
	Sensor_params []string
}

var cfg *config
var version = "0.1"

func main() {
	configPtr := flag.String("config", "server.json", "Path to the config file")
	flag.Parse()
	flag.Lookup("logtostderr").Value.Set("true")
	fmt.Println("Tempserver", version, "starting up")
	cfg = new(config)
	parseConfig(configPtr)
	startServer()
}

func parseConfig(location *string) {
	file, err := os.Open(*location)
	checkExit(err)

	decoder := json.NewDecoder(file)
	err = decoder.Decode(cfg)
	checkExit(err)
}

func checkExit(err error) {
	if err != nil {
		glog.Exit(err)
	}
}

func checkErr(err error) {
	if err != nil {
		glog.Error(err)
	}
}

func startServer() {
	glog.V(1).Infoln("Reading config...")

	glog.V(1).Infoln("Name", cfg.Name)
	glog.V(1).Infoln("Starting UDP server...")
	saddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", cfg.Ipv4_addr, cfg.Port))
	checkErr(err)
	sbind, err := net.ListenUDP("udp", saddr)
	checkErr(err)
	defer sbind.Close()

	buf := make([]byte, 1024)

	for {
		glog.V(1).Infoln("Listen loop")
		n, src, err := sbind.ReadFromUDP(buf)
		glog.V(1).Infof("ReceivedFrom %s\n", src)
		checkErr(err)
		command := temp.Command{}
		if err := proto.Unmarshal(buf[:n], &command); err != nil {
			glog.Warningln("Unmarshal error", err)
		}

		glog.V(1).Infof("Command: ", command.Command)
		if command.Command != temp.Command_REQUEST {
			glog.V(1).Infof("Non-Request received, ignoring: %s", command.Command)
			continue
		}

		glog.V(1).Infoln("Request received. Responding")
		sendTemp(command.Address.Ipv4, command.Address.Port)
	}
}

func sendTemp(address string, port int32) {
	glog.V(1).Infoln("Sending temperature to", address, port)

	s, err := net.DialUDP("udp4", nil, &net.UDPAddr{
		IP:   net.ParseIP(address),
		Port: int(port),
	})

	if err != nil {
		glog.Warningln("Error dialing", err)
		return
	}
	var t Sensor
	switch s := cfg.Sensor; s {
	case "tempAlert":
		t = &tempAlert{USB: cfg.Sensor_params[0]}
	case "ds18b20":
		t = &ds18b20_sensor{}
	case "fake_temp":
		desired, err  := strconv.ParseFloat(cfg.Sensor_params[0], 32)
		if err != nil {
			desired = 0.00;
		}

		t= &fake_temp{desired: float32(desired)}
	}
	err, temperature := t.getTemp()
	if err != nil {
		glog.V(1).Infoln("Temperature get error", err)
		return
	}
	response := temp.Command{
		Command:  temp.Command_REPLY,
		NodeName: cfg.Name,
		Temperature: &temp.Temperature{
			Temperature: temperature,
		},
	}

	data, err := proto.Marshal(&response)
	if err != nil {
		glog.V(1).Infoln("Error ", err)
		return
	}

	_, err = s.Write([]byte(data))
	if err != nil {
		glog.V(1).Infoln("Error sending payload")
	}
}

func (t *tempAlert) getTemp() (error, float32) {
	c := &serial.Config{Name: t.USB, Baud: 9600}

	s, err := serial.OpenPort(c)
	if err != nil {
		glog.Errorln(err)
	}
	// RESET
	n, err := s.Write([]byte("XP")) // Reset

	if err != nil {
		glog.Errorln(err)
	}
	buf := make([]byte, 128)

	n, err = s.Read(buf) // Discard

	if err != nil {
		glog.Errorln(err)
	}

	n, err = s.Write([]byte("R")) // Read

	if err != nil {
		glog.Errorln(err)
	}
	// Wait for the chip to read the temperature.
	time.Sleep(200 * time.Millisecond)
	n, err = s.Read(buf)

	if err != nil {
		glog.Errorln(err)
	}

	if len(buf[:n]) != 18 {
		glog.Errorln("Serial read is not 18 char")
	}

	var temperatureIndex int8
	if buf[15] == 0 && buf[16] == 0 && buf[17] == 0 {
		temperatureIndex = 0
	} else {
		temperatureIndex = 9
	}

	var temporary int32
	temporary = int32(buf[temperatureIndex]) | int32(buf[temperatureIndex+1])<<8

	var isNegative bool

	if (temporary & 0x8000) == 0x8000 {
		temporary &= 0x07ff
		isNegative = true
		temporary = 0x800 - temporary

	} else {
		isNegative = false
	}

	temporary &= 0x07ff

	celsius := float32(float32(temporary) / 16.0)
	if isNegative {
		celsius *= -1
	}
	glog.V(1).Infoln("Temperature:", celsius)
	return nil, celsius
}

func (t *ds18b20_sensor) getTemp() (error, float32) {
	sensors, err := ds18b20.Sensors()
	if err != nil {
		return err, 0.0
	}

	if len(sensors) != 1 {
		return errors.New("Expected 1 sensor"), 0.0
	}
	sensor := sensors[0]
	celsius, err := ds18b20.Temperature(sensor)
	if err != nil {
		return err, 0.0
	}
	return nil, float32(celsius)
}

func (t *fake_temp) getTemp() (error, float32) {
	return nil, t.desired
}