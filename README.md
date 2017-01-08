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
    > GOPATH=~/go
    > mkdir ~/go && cd $_
    > 
    ```
