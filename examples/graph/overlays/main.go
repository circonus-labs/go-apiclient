package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	api "github.com/circonus-labs/go-apiclient"
)

type OverlayList struct {
	OverlaySets *map[string]api.GraphOverlaySet `json:"overlay_sets,omitempty"` // GroupOverLaySets or null
}

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
	flag.StringVar(&cid, "cid", "", "graph cid e.g. --cid=123 or --cid=/graph/123 -- NOTE: graph will be EDITED make one explicitly for this test")
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

	if cid == "" {
		log.Fatal("--cid not set, required...create a temporary graph, it will be edited.")
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

	osFile := "graph_overlay_sets.json"
	data, err := ioutil.ReadFile(osFile)
	if err != nil {
		log.Fatalf("error reading %s (%s)", osFile, err)
	}

	var overlayList OverlayList
	if err := json.Unmarshal(data, &overlayList); err != nil {
		log.Fatal(err)
	}

	for osID, overlays := range *overlayList.OverlaySets {
		for overlayID, overlay := range overlays.Overlays {
			fmt.Printf("Testing overlay - ID: %s Type: %+v", overlayID, overlay.UISpecs.Type)
			g, err := client.FetchGraph(api.CIDType(&cid))
			if err != nil {
				log.Fatalf("\n\tERROR: getting (%s)", err)
			}
			newOverlaySet := map[string]api.GraphOverlaySet{
				osID: api.GraphOverlaySet{
					Title: "test",
					Overlays: map[string]api.GraphOverlay{
						overlayID: overlay,
					},
				},
			}
			g.OverlaySets = &newOverlaySet
			if _, err := client.UpdateGraph(g); err != nil {
				log.Fatalf("\n\tERROR: updating (%s)", err)
			}
			fmt.Printf(" -- SUCCESS\n")
		}
	}

	// remove the last overlay set
	g, err := client.FetchGraph(api.CIDType(&cid))
	if err != nil {
		log.Fatalf("getting (%s)", err)
	}
	g.OverlaySets = nil
	if _, err := client.UpdateGraph(g); err != nil {
		log.Fatalf("updating (%s)", err)
	}
}
