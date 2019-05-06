This class provides a first entry point to use the southbound client manager to connect to gNMI targets.

To run you need the following packages
``` 
go get https://github.com/openconfig/gnmi
go get github.com/openconfig/ygot/ygot

git clone https://github.com/openconfig/gnmi.git
```

The following assumes that you have started at least one docker emulator as per the guide here:
https://github.com/opennetworkinglab/onos-config/blob/master/tools/test/README.md 

To Execute
```
cd $GOPATH/src/github.com/opennetworkinglab/onos-config/southbound_main
go run southbound_main.go
```

if you want to change the Target address for now you woudl need to go and edit:

```
southbound_main.go from line 12 to 18
```