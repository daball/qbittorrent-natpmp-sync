# qBittorrent NAT-PMP Sync

A Go program that updates the NAT-PMP settings for qBittorrent Web UI client behind a ProtonVPN Wireguard connection, ensuring proper port forwarding through a router. This program was vibecoded overnight with Meta AI at the helm under my supervision. I have obviously enhanced the program a little but some of the core code was composed by Meta AI.

## Prerequisites

This program assumes you have a qBittorrent Web UI client configured with a Bittorrent listen port, a VPN router/gateway configured to connect to the Internet through a Wireguard client, that your VPN provider supports NAT-PMP for the Wireguard tunnel, and that you have either forwarded the ports through your gateway to the tunnel's interface or optionally configured NAT-PMP or UPnP on your router and client. At this point in your configuration you will think you've done everything you must in order to listen for traffic, but as it turns out, you need to query the tunnel's far gateway's NAT-PMP once per minute to get the external IP and port number and keep that in sync with the `announce_ip` and `announce_port` settings in the qBittorrent client, then reannounce to your trackers.

## Overview

This project provides a Dockerized solution for updating the NAT-PMP settings in qBittorrent, allowing for seamless port forwarding through a router when using a VPN Wireguard connection.
I'm not sure if it works in Docker as I'm using Podman Compose for my solution. It may also work for properly configured port forwarding over OpenVPN as well, assuming the remote OpenVPN supports NAT-PMP.

## Features

- Updates qBittorrent's `announce_ip` and `announce_port` settings based on the current NAT-PMP mapping.
- Works with ProtonVPN Wireguard connections. I'm using ProtonVPN.
- Dockerized for easy deployment and management.
- Compatible with routers like OpenWrt / OPNsense.

## Usage

1. Clone this repository and navigate to the project directory.
2. Create/edit `docker-compose.yml` file with the necessary configuration (see example below).
3. Run `docker compose up -d --build` or `podman compose up -d --build` to start the container in detached mode.
4. The program will periodically update the `announce_ip` and `announce_port` settings in qBittorrent to match the information returned from Wireguard far gateway's NAT-PMP query.

## Example docker-compose.yml

	services:
	  qbittorrent-natpmp-updater:
	    build: .
	    environment:
	      - WEBUI_BASE_URL=http://qbittorrent:8080
	      - WG_GATEWAY_IP=10.2.0.1
	      - WEBUI_USERNAME=your_qbittorrent_username
	      - WEBUI_PASSWORD=your_qbittorrent_password
	    restart: unless-stopped

## Configuration

The program requires the following environment variables:

- `WEBUI_BASE_URL`: The base URL. (e.g., `http://qbittorrent:8080`)
- `WG_GATEWAY_IP`: The far gateway IP for the Wireguard tunnel, as needed for the `natpmpc -g $WG_GATEWAY_IP` command. In the case of ProtonVPN with wireguard this host IP is `10.2.0.1`.
- `WEBUI_USERNAME`: The qBittorrent Web UI username (if configured).
- `WEBUI_PASSWORD`: The qBittorrent Web UI password (if configured).

## Building and Running in Docker

To build the Docker image, run `docker build -t natpmp-updater .` in the project directory. Or, create a `docker-compose.yml` file and run `docker compose up -d --build` to start the container.

## Building and Running in Podman

To build the Podman image, run `podman build -t natpmp-update -f Dockerfile` in the project directory. Or, create a `docker-compose.yml` file and run `podman compose up -d --build` to start the pod and the container.

## More Complete Example

In my scenario, I have created a `vpn` network in Podman, a macvlan type of network configuration, and it is attached to a separate network interface which is only attached to my VPN VLAN (network 10.2.1.0/24, router IP 10.2.1.1). That VPN VLAN is already configured with an OpenWrt router properly configured to tunnel through ProtonVPN and I have set up port forwarding to my qBittorrent static IP and listening port. The details of the `vpn` network are as follows for me:

	[
	     {
	          "name": "vpn",
	          "driver": "macvlan",
	          "network_interface": "enp6s19", // the interface you want to bridge to, which in my case is a separate virtual NIC on an isolated VLAN just for VPN traffic
	          "subnets": [
	               {
	                    "subnet": "10.2.1.0/24",
	                    "gateway": "10.2.1.1"
	               }
	          ],
	          "ipv6_enabled": false,
	          "internal": false,
	          "dns_enabled": false,
	          "ipam_options": {
	               "driver": "host-local"
	          }
	     }	
	]



Here's a compatible Docker Compose configuration:

	services:
	  qbittorrent:
	    image: lscr.io/linuxserver/qbittorrent:latest
	    container_name: qbittorrent
	    environment:
	      - PUID=1000
	      - PGID=1000
	      - TZ=Etc/UTC
	      - WEBUI_PORT=8080
	      - TORRENTING_PORT=6881
	    volumes:
	      - ./qbittorrent/appdata:/config:Z
	      - ./resolv.conf:/etc/resolv.conf:Z
	      - /mnt/media:/media:Z
	    networks:
	      vpn:
	        ipv4_address: 10.2.1.2
	    ports:
	      - 8080:8080
	      - 6881:6881
	      - 6881:6881/udp
	    restart: unless-stopped
	  natpmp-update:
	    build:
	      context: .
	    environment:
	      - WG_GATEWAY_IP=10.2.0.1
	      - WEBUI_BASE_URL=http://10.2.1.2:8080
	    networks:
	      vpn:
	        ipv4_address: 10.2.1.4
	    restart: unless-stopped
	networks:
	  vpn:
	    external: true
	    ipam:
	      config:
	        - subnet: 10.2.1.0/24
	          gateway: 10.2.1.1
	          dns:
	            - 1.1.1.1
	            - 1.0.0.1

## Standalone Operation

There are perhaps other ways to configure this. See the `Dockerfile` for details on how I build the Go program and then run it using the `entrypoint.sh` script, which reads the environment variables and transforms them into command line arguments.

## License

This project is licensed under the MIT License. See LICENSE for details.
