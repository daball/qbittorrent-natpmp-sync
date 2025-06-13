#!/bin/sh
if [ -n "${WEBUI_BASE_URL}" -a -n "${WG_GATEWAY_IP}" ]; then
  if [ -n "${WEBUI_USERNAME}" -a -n "${WEBUI_PASSWORD}" ]; then
    CMD="./update-natpmp -webui-base-url ${WEBUI_BASE_URL} -wg-gateway-ip ${WG_GATEWAY_IP} -username ${WEBUI_USERNAME} -password ${WEBUI_PASSWORD}"
  else
    CMD="./update-natpmp -webui-base-url ${WEBUI_BASE_URL} -wg-gateway-ip ${WG_GATEWAY_IP}"
  fi

  # Handle SIGINT and SIGTERM signals
  _term() {
    echo "[entrypoint.sh] Received terminate signal, forwarding to app..."
    kill -s SIGTERM $APP_PID
  }

  # Trap signals and forward them to app
  trap '_term' INT TERM

  # Start program in the background
  echo "[entrypoint.sh] Running command ${CMD}"
  ${CMD} &
  APP_PID=$!

  # Wait for program to finish
  wait $APP_PID
  EXIT_CODE=$?

  # Dump error code
  if [ $EXIT_CODE -ne 0 ]; then
    echo "[entrypoint.sh] Error running command ${CMD}, exit code ${EXIT_CODE}"
  fi
  exit $EXIT_CODE
else
  # Print usage
  echo "To use entry-point.sh, be sure to set environment first."
  echo "Required: WEBUI_BASE_URL (with no trailing slash)"
  echo "Required: WG_GATEWAY_IP (Proton VPN Wireguard client uses 10.2.0.1 as the far gateway.)"
  echo "Optionally required if set: WEBUI_USERNAME (qBittorrent default is admin.)"
  echo "Optionally required if set: WEBUI_PASSWORD (qBittorrent default is randomly generated.)"
  exit 1
fi
