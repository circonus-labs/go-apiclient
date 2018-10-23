package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	api "github.com/circonus-labs/go-apiclient"
)

func main() {
	var (
		apiKey   string
		apiApp   string
		apiDebug bool
		cid      string
	)
	flag.BoolVar(&apiDebug, "debug", false, "turn on debug messages")
	flag.StringVar(&apiKey, "key", "", "api token key")
	flag.StringVar(&apiApp, "app", "", "api token app name")
	flag.StringVar(&cid, "cid", "", "graph cid e.g. --cid=123 or --cid=/graph/123")
	flag.Parse()

	logger := log.New(os.Stdout, "", log.LstdFlags)

	if apiKey == "" {
		apiKey = os.Getenv("CIRCONUS_API_TOKEN")
		if apiKey == "" {
			log.Fatal("--key not used and CIRCONUS_API_TOKEN not set")
		}
	}

	if apiApp == "" {
		apiApp = os.Getenv("CIRCONUS_API_APP")
		if apiApp == "" {
			log.Fatal("--app not used and CIRCONUS_API_APP not set")
		}
	}

	if apiDebug {
		log.Printf(`[DEBUG] credentials: key="%s" app="%s"`, apiKey, apiApp)
	}

	client, err := api.New(&api.Config{
		TokenKey: apiKey,
		TokenApp: apiApp,
		Debug:    apiDebug,
		Log:      logger,
	})

	if err != nil {
		log.Fatal(err)
	}

	v, err := client.FetchGraph(api.CIDType(&cid))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%#v\n", v)
}
