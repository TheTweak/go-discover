package yc_test

import (
	"log"
	"os"
	"testing"

	discover "github.com/hashicorp/go-discover"
	"github.com/hashicorp/go-discover/provider/yc"
)

func TestAddrs(t *testing.T) {
	args := discover.Config{
		"provider":    "yc",
		"folder_id":   os.Getenv("YC_FOLDER_ID"),
		"label_name":  "foo",
		"label_value": "bar",
	}

	if args["folder_id"] == "" {
		t.Skip("YC folder_id missing")
	}

	p := &yc.Provider{}
	l := log.New(os.Stderr, "", log.LstdFlags)
	addrs, err := p.Addrs(args, l)
	if err != nil {
		t.Fatal(err)
	}
	if len(addrs) != 2 {
		t.Fatalf("bad: %v", addrs)
	}
}