#!/bin/sh
if [ -n "${WEBUI_BASE_URL}" -a -n "${WG_GATEWAY_IP}" ]; then
  if [ -n "${WEBUI_USERNAME}" -a -n "${WEBUI_PASSWORD}" ]; then
    CMD="./update-natpmp -webui-base-url \"${WEBUI_BASE_URL}\" -wg-gateway-ip \"${WG_GATEWAY_IP}\" -username \"${WEBUI_USERNAME}\" -password \"${WEBUI_PASSWORD}\""
  else
    CMD="./update-natpmp -webui-base-url \"${WEBUI_BASE_URL}\" -wg-gateway-ip \"${WG_GATEWAY_IP}\""
  fi
  echo "Running command ${CMD}"
  sh -c "${CMD}"
  if [ $? -ne 0 ]; then
    echo "Error running command ${CMD}"
    exit 1
  fi
else
  echo "To use entry-point.sh, be sure to set environment first."
  echo "Required: WEBUI_BASE_URL (with no trailing slash)"
  echo "Required: WG_GATEWAY_IP (Proton VPN Wireguard client uses 10.2.0.1 as the far gateway.)"
  echo "Optionally required if set: WEBUI_USERNAME (qBittorrent default is admin.)"
  echo "Optionally required if set: WEBUI_PASSWORD (qBittorrent default is randomly generated.)"
fi
