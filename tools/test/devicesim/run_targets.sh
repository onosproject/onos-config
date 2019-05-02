#!/bin/bash

if [ $SIM_MODE == 1 ]; 
then
    IPADDR=`ip route get 1.2.3.4 | grep dev | awk '{print $7}'` && \
    if [ "$GNMI_TARGET" != "localhost" ]; then \
      certs/generate_certs.sh $GNMI_TARGET > /dev/null 2>&1; \
      echo "Please add '"$IPADDR" "$GNMI_TARGET"' to /etc/hosts and access with gNMI client at "$GNMI_TARGET":"$GNMI_PORT; \
    else \
        echo "gNMI running on localhost:"$GNMI_PORT; \
    fi 
    sed -i -e "s/replace-device-name/"$GNMI_TARGET"/g" target_configs/typical_ofsw_config.json && \
    sed -i -e "s/replace-motd-banner/Welcome to gNMI service on "$GNMI_TARGET":"$GNMI_PORT"/g" target_configs/typical_ofsw_config.json \

    gnmi_target \
       -bind_address :$GNMI_PORT \
       -key $HOME/certs/$GNMI_TARGET.key \
       -cert $HOME/certs/$GNMI_TARGET.crt \
       -ca $HOME/certs/onfca.crt \
       -alsologtostderr \
       -config target_configs/typical_ofsw_config.json > /dev/null 2>&1; 
elif [ $SIM_MODE == 2 ]; 
then
    IPADDR=`ip route get 1.2.3.4 | grep dev | awk '{print $7}'` && \
    if [ "$GNOI_TARGET" != "localhost" ]; then \
      certs/generate_certs.sh $GNOI_TARGET > /dev/null 2>&1; \
      echo "Please add '"$IPADDR" "$GNOI_TARGET"' to /etc/hosts and access with gNOI client at "$GNOI_TARGET":"$GNOI_PORT; \
    else \
        echo "gNOI running on localhost:"$GNOI_PORT; 
    fi    	
    gnoi_target \
      -bind_address :$GNOI_PORT \
      -alsologtostderr;
elif [ $SIM_MODE == 3 ];
then
    IPADDR=`ip route get 1.2.3.4 | grep dev | awk '{print $7}'` && \
    if [ "$GNMI_TARGET" != "localhost" ]; then \
      certs/generate_certs.sh $GNMI_TARGET > /dev/null 2>&1; \
      echo "Please add '"$IPADDR" "$GNMI_TARGET"' to /etc/hosts and access with gNMI client at "$GNMI_TARGET":"$GNMI_PORT; \
    else \
        echo "gNMI running on localhost and port:"${GNMI_PORT}; 
    fi
    gnmi_target \
       -bind_address :$GNMI_PORT \
       -key $HOME/certs/$GNMI_TARGET.key \
       -cert $HOME/certs/$GNMI_TARGET.crt \
       -ca $HOME/certs/onfca.crt \
       -alsologtostderr \
       -config target_configs/typical_ofsw_config.json > /dev/null 2>&1 &
    
    if [ "$GNOI_TARGET" != "localhost" ]; then \
      certs/generate_certs.sh $GNOI_TARGET > /dev/null 2>&1; \
      echo "Please add '"$IPADDR" "$GNOI_TARGET"' to /etc/hosts and access with gNOI client at "$GNOI_TARGET":"$GNOI_PORT; \ 
    else \
        echo "gNOI running on localhost:"$GNOI_PORT; 
    fi 
    gnoi_target \
      -bind_address :$GNOI_PORT \
      -alsologtostderr;
fi