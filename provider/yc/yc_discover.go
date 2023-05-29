// Package yc provides node discovery for Yandex Cloud.
package yc

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"time"
	"strings"
	"context"

	"github.com/yandex-cloud/go-genproto/yandex/cloud/compute/v1"	
	ycsdk "github.com/yandex-cloud/go-sdk"
)

type Provider struct{
	iamToken *string
	iamTokenCreateTime time.Time
}

func (p *Provider) Help() string {
	return `Yandex Cloud:

    provider:    "yc"
    folder_id:   The Yandex Cloud folder ID
    label_name:  The label name to filter on
    label_value: The label value to filter on
    addr_type:   "private_v4", "public_v4" or "public_v6". Defaults to "private_v4"    

    The only required IAM role is 'compute.viewer', code will lazy generate new IAM token every hour using yc CLI.
`
}

func (p *Provider) Addrs(args map[string]string, l *log.Logger) ([]string, error) {
	if args["provider"] != "yc" {
		return nil, fmt.Errorf("discover-yc: invalid provider " + args["provider"])
	}

	if l == nil {
		l = log.New(ioutil.Discard, "", 0)
	}

	folderId := args["folder_id"]
	labelName := args["label_name"]
	labelValue := args["label_value"]
	addrType := args["addr_type"]
	iamToken := args["iam_token"]

	if p.iamToken == nil {
		// create new IAM token
		p.iamToken = createIAMToken()
	} else {
		// check if current token is more than 1 hour old		
		if et := p.iamTokenCreateTime.Add(time.Hour * 1); time.Now().After(et) {
			// create new IAM token
			p.iamToken = createIAMToken()
		}
	}

	ctx := context.Background()

	sdk, err := ycsdk.Build(ctx, ycsdk.Config{
		Credentials: ycsdk.NewIAMTokenCredentials(p.iamToken),
	})
	if err != nil {
		l.Println("[ERROR] discover-yc: Failed to create yc SDK client")
		return nil, err
	}
	addrs, err := getInstancesAddrs(ctx, sdk, *folderID, "auto-join", "master-v1")
	if err != nil {
		l.Println("[ERROR] discover-yc: failed to get instances addrs")		
		return nil, err
	}
	return addrs, nil
}

func getInstancesAddrs(ctx context.Context, sdk *ycsdk.SDK, folderID string, labelName string, labelValue string) ([]string, error) {
	result := make([]string, 0)
	var pageToken string
	for {
		request := &compute.ListInstancesRequest {
			FolderId: folderID,
			PageSize: 64,
			PageToken: pageToken,
		}

		resp, err := sdk.Compute().Instance().List(ctx, request)

		if err != nil {
			return result, err
		}

		for _, instance := range resp.Instances {
			if lv, ok := instance.Labels[labelName]; ok && lv == labelValue {
				if len(instance.NetworkInterfaces) == 1 {
					result = append(result, instance.NetworkInterfaces[0].PrimaryV4Address.Address)
				}	
			}
		}

		pageToken = resp.NextPageToken

		if len(pageToken) == 0 {
			break
		}
	}		
	return result, nil
}

func createIAMToken(l *log.Logger) (*string, error) {
	cmd := exec.Command("yc", "iam", "create-token")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		l.Println("[ERROR] discover-yc: Failed to create stdout pipe for yc iam create-token command")
		return nil, err
	}

	if err = cmd.Start(); err != nil {
		return nil, err
	}

	buf := new(strings.Builder)
	if _, err = io.Copy(buf, stdout); err != nil {
		return nil, err
	}
	
	return buf.String()
}