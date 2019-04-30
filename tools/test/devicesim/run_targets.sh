#!/bin/bash

if [ $SIM_MODE == 1 ]; 
then
    IPADDR=`ip route get 1.2.3.4 | grep dev | awk '{print $7}'` && \
    if [ "$HOST_TARGET" != "localhost" ]; then \
      echo "Please add '"$IPADDR" "$HOST_TARGET"' to /etc/hosts and access with gNMI client at "$HOST_TARGET":"$GNMI_PORT; \
    else \
        echo "gNMI running on localhost:"$GNMI_PORT; \
    fi 
    sed -i -e "s/replace-device-name/"$HOST_TARGET"/g" target_configs/typical_ofsw_config.json && \
    sed -i -e "s/replace-motd-banner/Welcome to gNMI service on "$HOST_TARGET":"$GNMI_PORT"/g" target_configs/typical_ofsw_config.json \

    gnmi_target \
       -bind_address :$GNMI_PORT \
       -key $HOME/certs/$HOST_TARGET.key \
       -cert $HOME/certs/$HOST_TARGET.crt \
       -ca $HOME/certs/onfca.crt \
       -alsologtostderr \
       -config target_configs/typical_ofsw_config.json > /dev/null 2>&1; 
elif [ $SIM_MODE == 2 ]; 
then
    IPADDR=`ip route get 1.2.3.4 | grep dev | awk '{print $7}'` && \
    if [ "$HOST_TARGET" != "localhost" ]; then \
      echo "Please add '"$IPADDR" "$HOST_TARGET"' to /etc/hosts and access with gNOI client at "$HOST_TARGET":"$GNOI_PORT; \
    else \
        echo "gNOI running on localhost:"$GNOI_PORT; 
    fi    	
    gnoi_target \
      -bind_address :$GNOI_PORT \
      -alsologtostderr;
elif [ $SIM_MODE == 3 ];
then
    IPADDR=`ip route get 1.2.3.4 | grep dev | awk '{print $7}'` && \
    if [ "$HOST_TARGET" != "localhost" ]; then \
      echo "Please add '"$IPADDR" "$HOST_TARGET"' to /etc/hosts and access with gNMI/gNOI clients at "$HOST_TARGET":"$GNMI_PORT":"$GNOI_PORT; \
    else \
        echo "gNMI running on localhost and port:"${GNMI_PORT}; 
        echo "gNOI running on localhost:"$GNOI_PORT; 
    fi
    gnmi_target \
       -bind_address :$GNMI_PORT \
       -key $HOME/certs/$HOST_TARGET.key \
       -cert $HOME/certs/$HOST_TARGET.crt \
       -ca $HOME/certs/onfca.crt \
       -alsologtostderr \
       -config target_configs/typical_ofsw_config.json > /dev/null 2>&1 &
    
    gnoi_target \
      -bind_address :$GNOI_PORT \
      -alsologtostderr;
fi