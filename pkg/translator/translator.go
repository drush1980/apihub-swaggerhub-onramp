package translator

import (
	"apigee-swaggerhub-plugin/pkg/swaggerhub"
	"log"
	"strings"
	"time"

	apihubpb "cloud.google.com/go/apihub/apiv1/apihubpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/yaml.v3"
)

type openApiSpec struct {
	Info struct {
		Contact struct {
			Name  string `yaml:"name"`
			Email string `yaml:"email"`
		} `yaml:"contact"`
	} `yaml:"info"`
	ExternalDocs struct {
		URL string `yaml:"url"`
	} `yaml:"externalDocs"`
}

func Translate(apis []swaggerhub.APISummary, apiSpecs map[string][]byte) []*apihubpb.APIMetadata {
	var apiMetadataList []*apihubpb.APIMetadata

	for _, api := range apis {
		log.Printf("Translating API: %s", api.Name)

		now := timestamppb.Now()
		createdTime := now
		modifiedTime := now

		for _, prop := range api.Properties {
			switch prop.Type {
			case "X-Created":
				if t, err := time.Parse(time.RFC3339, prop.Value); err == nil {
					createdTime = timestamppb.New(t)
				}
			case "X-Modified":
				if t, err := time.Parse(time.RFC3339, prop.Value); err == nil {
					modifiedTime = timestamppb.New(t)
				}
			}
		}

		// For now, we use a default version "v1" if we can't find it.
		// In a real scenario, we would extract this from the API summary or spec.
		versionID := "v1"
		for _, prop := range api.Properties {
			if prop.Type == "X-Version" {
				versionID = prop.Value
				break
			}
		}

		url := ""
		for _, prop := range api.Properties {
			if prop.Type == "Swagger" {
				url = prop.URL
				break
			}
		}

		apiID := api.Name
		if url != "" {
			parts := strings.Split(strings.TrimRight(url, "/"), "/")
			if len(parts) >= 2 {
				apiID = parts[len(parts)-2]
			}
		}

		var owner *apihubpb.Owner
		var doc *apihubpb.Documentation
		var specs []*apihubpb.SpecMetadata
		if specContent, ok := apiSpecs[url]; ok {
			var spec openApiSpec
			if err := yaml.Unmarshal(specContent, &spec); err == nil {
				if spec.Info.Contact.Email != "" {
					owner = &apihubpb.Owner{
						Email:       spec.Info.Contact.Email,
						DisplayName: spec.Info.Contact.Name,
					}
				}
				if spec.ExternalDocs.URL != "" {
					doc = &apihubpb.Documentation{
						ExternalUri: spec.ExternalDocs.URL,
					}
				}
			}

			specs = []*apihubpb.SpecMetadata{
				{
					Spec: &apihubpb.Spec{
						DisplayName: "openapi.yaml",
						Contents: &apihubpb.SpecContents{
							Contents: specContent,
							MimeType: "text/yaml",
						},
						SpecType: &apihubpb.AttributeValues{
							Value: &apihubpb.AttributeValues_EnumValues{
								EnumValues: &apihubpb.AttributeValues_EnumAttributeValues{
									Values: []*apihubpb.Attribute_AllowedValue{
										{
											Id: "openapi",
										},
									},
								},
							},
						},
					},
					OriginalId:         "openapi.yaml",
					OriginalCreateTime: createdTime,
					OriginalUpdateTime: modifiedTime,
				},
			}
		}

		metadata := &apihubpb.APIMetadata{
			Api: &apihubpb.Api{
				DisplayName:   api.Name,
				Description:   api.Description,
				Owner:         owner,
				Documentation: doc,
				Fingerprint:   apiID,
				ApiStyle: &apihubpb.AttributeValues{
					Value: &apihubpb.AttributeValues_EnumValues{
						EnumValues: &apihubpb.AttributeValues_EnumAttributeValues{
							Values: []*apihubpb.Attribute_AllowedValue{
								{
									Id: "rest",
								},
							},
						},
					},
				},
			},
			OriginalId:         apiID,
			OriginalCreateTime: createdTime,
			OriginalUpdateTime: modifiedTime,
			Versions: []*apihubpb.VersionMetadata{
				{
					Version: &apihubpb.Version{
						DisplayName:   versionID,
						Documentation: doc,
					},
					OriginalId:         versionID,
					OriginalCreateTime: createdTime,
					OriginalUpdateTime: modifiedTime,
					Specs:              specs,
				},
			},
		}
		apiMetadataList = append(apiMetadataList, metadata)
	}

	return apiMetadataList
}
