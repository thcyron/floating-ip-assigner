package main

import (
	"context"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/pkg/errors"
)

const (
	timeout       = 10 * time.Second
	retryDelay    = 10 * time.Second
	checkInterval = 60 * time.Second
)

func main() {
	token := os.Getenv("HCLOUD_TOKEN")
	if token == "" {
		log.Fatalln("no token specified in HCLOUD_TOKEN")
	}

	floatingIPID, err := strconv.Atoi(os.Getenv("HCLOUD_FLOATING_IP_ID"))
	if err != nil {
		log.Fatalln("no or invalid Floating IP ID specified in HCLOUD_FLOATING_IP_ID")
	}

	client := hcloud.NewClient(
		hcloud.WithToken(token),
	)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	floatingIP, _, err := client.FloatingIP.GetByID(ctx, floatingIPID)
	if err != nil {
		log.Fatalln(errors.Wrap(err, "failed to get Floating IP from API"))
	}
	serverID, err := getInstanceID(ctx)
	if err != nil {
		log.Fatalln(errors.Wrap(err, "failed to get instance ID from metadata service"))
	}
	server, _, err := client.Server.GetByID(ctx, serverID)
	if err != nil {
		log.Fatalln(errors.Wrap(err, "failed to get server from API"))
	}
	cancel()

	for {
		ctx, cancel = context.WithTimeout(context.Background(), timeout)
		assigned, err := check(ctx, client, server, floatingIP)
		cancel()

		if err != nil {
			log.Println(errors.Wrap(err, "check failed"))
			log.Printf("retrying in %s...\n", retryDelay)
			time.Sleep(retryDelay)
		} else {
			if assigned {
				log.Printf("Floating IP assigned to %d\n", server.ID)
			}
			time.Sleep(checkInterval)
		}
	}
}

func check(ctx context.Context, client *hcloud.Client, server *hcloud.Server, floatingIP *hcloud.FloatingIP) (bool, error) {
	fip, _, err := client.FloatingIP.GetByID(ctx, floatingIP.ID)
	if err != nil {
		return false, errors.Wrap(err, "failed to get Floating IP from API")
	}

	if fip.Server == nil || fip.Server.ID != server.ID {
		if fip.Server == nil {
			log.Printf("Floating IP not assigned to any server; assigning to %d...\n", server.ID)
		} else {
			log.Printf("Floating IP assigned to %d; assigning to %d...\n", fip.Server.ID, server.ID)
		}
		assigned, err := assign(client, server, fip)
		if err != nil {
			return false, errors.Wrap(err, "failed to assign Floating IP")
		}
		return assigned, nil
	}

	return false, nil
}

func assign(client *hcloud.Client, server *hcloud.Server, floatingIP *hcloud.FloatingIP) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	action, _, err := client.FloatingIP.Assign(ctx, floatingIP, server)
	if err != nil {
		return false, err
	}

	_, errCh := client.Action.WatchProgress(ctx, action)
	if err := <-errCh; err != nil {
		return false, err
	}
	return true, nil
}

func getInstanceID(ctx context.Context) (int, error) {
	req, err := http.NewRequest(http.MethodGet, "http://169.254.169.254/2009-04-04/meta-data/instance-id", nil)
	if err != nil {
		return 0, err
	}
	req = req.WithContext(ctx)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	return strconv.Atoi(string(body))
}
