# Device Simulator

This is a docker VM that runs gNMI/gNOI implementation supporting openconfig models.

Inspired by https://github.com/faucetsdn/gnmi 

Everything below assumes you are in the __devicesim__ directory

**Note**: The device simulator can operate in three modes that allows you to create devices that can run gNMI target, gNOI target, and both gNMI and gNOI targets simultaneously. You can control the mode using **SIM_MODE** enviroment variable in the docker-compose file. 

You can access to the user manuals for each simulator using the following links: 

[gNMI User Manual](gnmi_user_manual.md)

[gNOI User Manual](gnoi_user_manual.md)

