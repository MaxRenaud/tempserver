TempServer
=

Intro
-
TempServer is a Client/Server application that can grab temperature from many remote sensors on the same network
and present them in an easy to read format.
```
> ./c
Attic: 15.35
Garage: 19.67
```
Installation
-
1. Compile the server on a raspberry pi
    1. Install go
    ```
    > apt-get install golang
    ```
    2. Compile
    ```
    # Assuming a functional $GOPATH
    > go get -u github.com/maxrenaud/tempserver/server # For the server
    > go get -u github.com/maxrenaud/tempserver/server # For the client
    ```
2. Configure
    1.  Copy the sample json config from github.com/maxrenaud/tempserver/server.json.sample to an accessible location 
    like /etc/tempserver.json. For the client.json.sample, I recommend puttint it in your home directory
    2. Edit the copied file to match your desired configuration
3. Test
    1. Launch the server
    ```
    > server -config="/etc/tempserver.json"
    ```
    2. In a different terminal, try the client
    ```
    > client -config="~/client.json" # -config not required if client.json is in the current directory
    ```
4. Configure the server to start at launch (Systemd)
    1. Copy the tempserver.service file to /etc/systemd/system
    ```
    > sudo cp tempserver.service /etc/systemd/system
    ```
    2. Edit the file and change the ExecStart variable to match where you installed the server
    3. Configure the service to come up at boot time
    ```
    > sudo systemctl enable tempserver
    ```
    4. Reboot the Pi. Alternatively, you can manually start the service
    ```
    > sudo systemctl start tempserver
    ```
    
Details
-
In server.json, the sensor and sensor_params are currently as follow:
1. sensor 
    1. "tempAlert" for the [TempAlert](http://shop.tempalert.com/usb-temperature-sensor.aspx).
    2. "ds18b20" for the [ds18b20 chip](https://cdn-learn.adafruit.com/downloads/pdf/adafruits-raspberry-pi-lesson-11-ds18b20-temperature-sensing.pdf)
    3. "fake_temp" for a fake module that always reports the same temperature.
    
2. sensor_param. There are the string parameters to pass on to the sensor module
    1. "tempAlert" takes one parameter, the location of the tempAlert. E.g: /dev/ttyUSB0
    2. "ds18b20" does not have any parameters
    3. "fake_temp" takes one parameter, the desired temperature output