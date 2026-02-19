package aggregate

import (
	"hyperwapp/model"
	"sort"
)

// AggregatedDomain represents detections grouped by domain.
type AggregatedDomain struct {
	Domain     string
	URLs       []string
	Detections []model.Detection
}

// AggregateByDomain groups a slice of detections by their domain.
func AggregateByDomain(allDetections []model.Detection) []AggregatedDomain {
	domainMap := make(map[string]map[string]struct{})  // domain -> URL -> struct{} (for unique URLs)
	detectionMap := make(map[string][]model.Detection) // domain -> []Detection

	for _, d := range allDetections {
		if _, ok := domainMap[d.Domain]; !ok {
			domainMap[d.Domain] = make(map[string]struct{})
		}
		domainMap[d.Domain][d.URL] = struct{}{}
		detectionMap[d.Domain] = append(detectionMap[d.Domain], d)
	}

	var aggregated []AggregatedDomain
	for domain, urlsMap := range domainMap {
		var urls []string
		for url := range urlsMap {
			urls = append(urls, url)
		}
		sort.Strings(urls) // Sort URLs for consistent output

		aggregated = append(aggregated, AggregatedDomain{
			Domain:     domain,
			URLs:       urls,
			Detections: detectionMap[domain],
		})
	}

	// Sort aggregated domains by name
	sort.Slice(aggregated, func(i, j int) bool {
		return aggregated[i].Domain < aggregated[j].Domain
	})

	return aggregated
}
