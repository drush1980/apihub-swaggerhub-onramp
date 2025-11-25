package main

import (
	"apigee-swaggerhub-plugin/pkg/apigee"
	"apigee-swaggerhub-plugin/pkg/swaggerhub"
	"apigee-swaggerhub-plugin/pkg/translator"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	apihubpb "cloud.google.com/go/apihub/apiv1/apihubpb"
)

func main() {
	http.HandleFunc("/sync", syncHandler)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Listening on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func syncHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	pluginInstanceId := r.URL.Query().Get("plugin_instance")
	if pluginInstanceId == "" {
		http.Error(w, "Missing plugin_instance query parameter", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	apigeeClient, err := apigee.NewClient(ctx)
	if err != nil {
		log.Printf("Failed to create API hub client: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// instance, err := apigeeClient.GetPluginInstance(ctx, fmt.Sprintf("projects/%s/locations/%s/plugins/swaggerhub-plugin/instances/%s", getProjectID(), getRegion(), pluginInstanceId))
	// if err != nil {
	// 	log.Printf("Failed to get plugin instance: %v", err)
	// 	http.Error(w, "Internal server error", http.StatusInternalServerError)
	// 	return
	// }

	// Extract Swaggerhub config from instance.AdditionalConfig
	// Assuming keys are "swaggerhub_owner" and "swaggerhub_api_key"
	// owner := ""
	// apiKey := ""
	// if instance.AdditionalConfig != nil {
	// 	if val, ok := instance.AdditionalConfig["swaggerhub_owner"]; ok {
	// 		owner = val.StringValue
	// 	}
	// 	if val, ok := instance.AdditionalConfig["swaggerhub_api_key"]; ok {
	// 		apiKey = val.StringValue
	// 	}
	// }

	owner := os.Getenv("SWAGGERHUB_OWNER")
	apiKey := os.Getenv("SWAGGERHUB_API_KEY")

	if owner == "" {
		log.Printf("Missing swaggerhub_owner in plugin config")
		http.Error(w, "Invalid plugin configuration", http.StatusBadRequest)
		return
	}

	log.Printf("Processing sync...")

	shClient := swaggerhub.NewClient("", apiKey)
	apis, err := shClient.ListAPIs(owner)
	if err != nil {
		log.Printf("Failed to list APIs from Swaggerhub: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	apiSpecs := make(map[string][]byte)
	for _, api := range apis {
		for _, prop := range api.Properties {
			if prop.Type == "Swagger" {
				spec, err := shClient.GetAPISpec(prop.URL)
				if err != nil {
					log.Printf("Failed to fetch spec for %s: %v", api.Name, err)
					continue
				}
				apiSpecs[prop.URL] = spec
				break
			}
		}
	}

	apiMetadata := translator.Translate(apis, apiSpecs)

	parent := fmt.Sprintf("projects/%s/locations/%s", os.Getenv("GOOGLE_CLOUD_PROJECT"), os.Getenv("GOOGLE_CLOUD_REGION"))
	req := &apihubpb.CollectApiDataRequest{
		Location:       parent,
		PluginInstance: fmt.Sprintf("projects/%s/locations/%s/plugins/swaggerhub-plugin/instances/%s", os.Getenv("GOOGLE_CLOUD_PROJECT"), os.Getenv("GOOGLE_CLOUD_REGION"), pluginInstanceId),
		ActionId:       "sync-action",
		CollectionType: apihubpb.CollectionType_COLLECTION_TYPE_UPSERT,
		ApiData: &apihubpb.ApiData{
			Data: &apihubpb.ApiData_ApiMetadataList{
				ApiMetadataList: &apihubpb.ApiMetadataList{
					ApiMetadata: apiMetadata,
				},
			},
		},
	}

	_, err = apigeeClient.CollectApiData(ctx, parent, req)
	if err != nil {
		log.Printf("Failed to collect API data: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// w.Header().Set("Content-Type", "application/json")
	// jsonResp, err := protojson.Marshal(collectResp)
	// if err != nil {
	// 	log.Printf("Failed to marshal response: %v", err)
	// 	return
	// }
	// log.Printf("Response: %s", string(jsonResp))

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Success"))
	log.Printf("Sync completed successfully")
}
