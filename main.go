package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
        "net/url"
	"os/exec"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type CmdArgs struct {
	webuiBaseUrl *string
	wgGatewayIP *string
	username *string
	password *string
	sleepTime *uint
}

func main() {
	ca := CmdArgs {
		webuiBaseUrl: flag.String("webui-base-url", "", "qBittorrent Web UI base URL"),
        	wgGatewayIP: flag.String("wg-gateway-ip", "", "Wireguard far gateway IP address"),
	        username: flag.String("username", "", "qBittorrent Web UI username"),
        	password: flag.String("password", "", "qBittorrent Web UI password"),
	        sleepTime: flag.Uint("sleep-time", 30 /*seconds*/, "Interval (integer in seconds) to run NAT-PMP") }
	flag.Parse()

	if *ca.webuiBaseUrl == "" {
		log.Fatal("Web UI base URL is required")
	}
	if *ca.wgGatewayIP == "" {
		log.Fatal("Wireguard gateway IP address is required")
	}

        ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
        defer stop()

        fmt.Println("Waiting for signal...")
	mainLoop(ctx, ca)
        fmt.Println("Shutdown complete")
}

func mainLoop(ctx context.Context, ca CmdArgs) {
        sleepingText := fmt.Sprintf("Sleeping for %d seconds.", *ca.sleepTime)

	for {
		select {
		case <-ctx.Done():
			fmt.Println("Received signal, exiting...")
			return
		default:
			doMainLoopWork(ca)
	                log.Print(sleepingText);
        	        time.Sleep(time.Duration(*ca.sleepTime) * time.Second)
		}
	}
}

func doMainLoopWork(ca CmdArgs) {
	currentAnnounceIP, currentAnnouncePort, currentListenPort, err := getCurrentPreferences(*ca.webuiBaseUrl, *ca.username, *ca.password)
	if err != nil {
		log.Printf("Error getting current preferences: %s", err)
		return
	}

	log.Printf("Running natpmpc -g %s -a 1 %d TCP 60", *ca.wgGatewayIP, currentListenPort);
	tcpOutput, err := exec.Command("natpmpc", "-g", *ca.wgGatewayIP, "-a", "1", fmt.Sprintf("%d", currentListenPort), "TCP", "60").CombinedOutput()
	if err != nil {
		log.Printf("Error running natpmpc for TCP: %s", tcpOutput)
		return
	}

        log.Printf("Running natpmpc -g %s -a 1 %d UDP 60", *ca.wgGatewayIP, currentListenPort);
	udpOutput, err := exec.Command("natpmpc", "-g", *ca.wgGatewayIP, "-a", "1", fmt.Sprintf("%d", currentListenPort), "UDP", "60").CombinedOutput()
	if err != nil {
		log.Printf("Error running natpmpc for UDP: %s", udpOutput)
		return
	}

	publicPortTcp, err := parsePublicPort(tcpOutput)
	if err != nil {
		log.Printf("Error parsing public port from TCP output: %s", err)
		return
	}
        publicIPTcp, err := parsePublicIP(tcpOutput)
        if err != nil {
                log.Printf("Error parsing public IP address from TCP output: %s", err)
		return
        }

	publicPortUdp, err := parsePublicPort(udpOutput)
	if err != nil {
		log.Printf("Error parsing public port from UDP output: %s", err)
		return
	}

	publicPort := publicPortTcp
	publicIP := publicIPTcp

	if publicPortUdp != publicPortTcp {
		log.Printf("Warning: public ports for TCP and UDP do not match: TCP=%d, UDP=%d", publicPortTcp, publicPortUdp)
		publicPort = publicPortUdp // Use the last returned public port
	}

	if publicPort != currentAnnouncePort {
		log.Printf("Updating announce port from %d to %d", currentAnnouncePort, publicPort)
		err = updateAnnouncePort(*ca.webuiBaseUrl, *ca.username, *ca.password, publicPort)
		if err != nil {
			log.Printf("Error updating announce port: %s", err)
		} else {
		        _, newAnnouncePort, _, err := getCurrentPreferences(*ca.webuiBaseUrl, *ca.username, *ca.password)
		        if err != nil {
		            log.Printf("Error getting current preferences after update: %s", err)
		        } else if newAnnouncePort == publicPort {
		            log.Printf("Announce port updated successfully to %d", publicPort)
		        } else {
		            log.Printf("Announce port update failed: expected %d, got %d", publicPort, newAnnouncePort)
		        }
		}
	} else {
		log.Print("There was no change needed to the announce_port. Leaving announce_port the same.")
	}

        if publicIP != currentAnnounceIP {
        	log.Printf("Updating announce IP from %s to %s", currentAnnounceIP, publicIP)
                err = updateAnnounceIP(*ca.webuiBaseUrl, *ca.username, *ca.password, publicIP)
                if err != nil {
                	log.Printf("Error updating announce IP: %s", err)
                } else {
                        newAnnounceIP, _, _, err := getCurrentPreferences(*ca.webuiBaseUrl, *ca.username, *ca.password)
                        if err != nil {
                	        log.Printf("Error getting current preferences after update: %s", err)
                        } else if newAnnounceIP == publicIP {
                        	log.Printf("Announce IP updated successfully to %s", publicIP)
                	} else {
                                log.Printf("Announce IP update failed: expected %s, got %s", publicIP, newAnnounceIP)
                        }
                }
        } else {
        	log.Print("There was no change needed to the announce_ip. Leaving announce_ip the same.")
        }
}

//parses natpmpc output for "Public IP address : (\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})"
func parsePublicIP(output []byte) (string, error) {
    re := regexp.MustCompile(`Public IP address : (\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})`)
    matches := re.FindSubmatch(output)
    if len(matches) != 2 {
        return "", errors.New("failed to parse public IP address from output")
    }
    return string(matches[1]), nil
}

//parses natpmpc output for "Mapped public port (\d+)" expression
func parsePublicPort(output []byte) (uint, error) {
	re := regexp.MustCompile(`Mapped public port (\d+)`)
	matches := re.FindSubmatch(output)
	if len(matches) != 2 {
		return 0, errors.New("failed to parse public port from output")
	}
	port, err := strconv.Atoi(string(matches[1]))
	if err != nil {
		return 0, fmt.Errorf("invalid port number: %w", err)
	}
	return uint(port), nil
}

func getCurrentPreferences(baseUrl string, username string, password string) (string, uint, uint, error) {
	getUrl := fmt.Sprintf("%s/api/v2/app/preferences", baseUrl)
        log.Printf("Getting current preferences from qBittorrent at endpoint %s", getUrl);
	client := &http.Client{}
	req, err := http.NewRequest("GET", getUrl, nil)
	if err != nil {
		return "", 0, 0, err
	}

	if username != "" && password != "" {
		req.SetBasicAuth(username, password)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", 0, 0, err
	}
	defer resp.Body.Close()

	var prefs map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&prefs)
	if err != nil {
		return "", 0, 0, err
	}

	announceIP, ok := prefs["announce_ip"].(string)
	if !ok {
		return "", 0, 0, errors.New("announce_ip not found or invalid type")
	}

	announcePort, ok := prefs["announce_port"].(float64)
	if !ok {
		return "", 0, 0, errors.New("announce_port not found or invalid type")
	}

	listenPort, ok := prefs["listen_port"].(float64)
	if !ok {
		return "", 0, 0, errors.New("listen_port not found or invalid type")
	}

	log.Printf("qBittorrent current preferences: listen_port=%d announce_ip=%s announce_port=%d", uint(listenPort), string(announceIP), uint(announcePort));

	return string(announceIP), uint(announcePort), uint(listenPort), nil
}

func getAllCurrentPreferences(baseUrl string, username string, password string) (map[string]interface{}, error) {
	getUrl := fmt.Sprintf("%s/api/v2/app/preferences", baseUrl)
	log.Printf("Getting current preferences from qBittorrent at endpoint %s", getUrl)
	client := &http.Client{}
	req, err := http.NewRequest("GET", getUrl, nil)
	if err != nil {
        	return nil, err
	}

	if username != "" && password != "" {
        	req.SetBasicAuth(username, password)
	}

	resp, err := client.Do(req)
	if err != nil {
	        return nil, err
	}
	defer resp.Body.Close()

	var prefs map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&prefs)
	if err != nil {
	        return nil, err
	}

	return prefs, nil
}

func updateAnnouncePort(baseUrl string, username string, password string, port uint) error {
    // Fetch the current preferences
    prefs, err := getAllCurrentPreferences(baseUrl, username, password)
    if err != nil {
        return err
    }

    // Update the announce_port setting
    prefs["announce_port"] = float64(port)

    // Marshal the preferences to JSON
    jsonBytes, err := json.Marshal(prefs)
    if err != nil {
        return err
    }

    // URL-encode the JSON payload
    encodedPayload := url.Values{
        "json": {string(jsonBytes)},
    }

    // Create a new request
    updateUrl := fmt.Sprintf("%s/api/v2/app/setPreferences", baseUrl)
    req, err := http.NewRequest("POST", updateUrl, strings.NewReader(encodedPayload.Encode()))
    if err != nil {
        return err
    }

    // Set the Content-Type header
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

    // Set the basic auth credentials
    if username != "" && password != "" {
        req.SetBasicAuth(username, password)
    }

    // Send the request
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    return nil
}

func updateAnnounceIP(baseUrl string, username string, password string, ip string) error {
    // Fetch the current preferences
    prefs, err := getAllCurrentPreferences(baseUrl, username, password)
    if err != nil {
        return err
    }

    // Update the announce_port setting
    prefs["announce_ip"] = string(ip)

    // Marshal the preferences to JSON
    jsonBytes, err := json.Marshal(prefs)
    if err != nil {
        return err
    }

    // URL-encode the JSON payload
    encodedPayload := url.Values{
        "json": {string(jsonBytes)},
    }

    // Create a new request
    updateUrl := fmt.Sprintf("%s/api/v2/app/setPreferences", baseUrl)
    req, err := http.NewRequest("POST", updateUrl, strings.NewReader(encodedPayload.Encode()))
    if err != nil {
        return err
    }

    // Set the Content-Type header
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

    // Set the basic auth credentials
    if username != "" && password != "" {
        req.SetBasicAuth(username, password)
    }

    // Send the request
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    return nil
}
