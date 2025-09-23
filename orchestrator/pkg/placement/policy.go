package placement

import (
	"fmt"
	"math"
	"sort"
	"time"
)

// LatencyAwarePlacementPolicy implements placement based on latency and resource requirements
type LatencyAwarePlacementPolicy struct {
	metricsProvider MetricsProvider
	// Weights for scoring components (should sum to 1.0)
	weights PlacementWeights
}

// PlacementWeights defines relative importance of different factors
type PlacementWeights struct {
	Latency     float64 // Weight for latency matching
	Resources   float64 // Weight for resource availability
	Throughput  float64 // Weight for throughput capability
	CloudType   float64 // Weight for cloud type preference
	Utilization float64 // Weight for current utilization
}

// DefaultWeights returns default placement weights
func DefaultWeights() PlacementWeights {
	return PlacementWeights{
		Latency:     0.25,
		Resources:   0.25,
		Throughput:  0.15,
		CloudType:   0.25, // Increased to give more weight to cloud type matching
		Utilization: 0.10,
	}
}

// NewLatencyAwarePlacementPolicy creates a new latency-aware placement policy
func NewLatencyAwarePlacementPolicy(provider MetricsProvider) *LatencyAwarePlacementPolicy {
	return &LatencyAwarePlacementPolicy{
		metricsProvider: provider,
		weights:         DefaultWeights(),
	}
}

// NewLatencyAwarePlacementPolicyWithWeights creates policy with custom weights
func NewLatencyAwarePlacementPolicyWithWeights(provider MetricsProvider, weights PlacementWeights) *LatencyAwarePlacementPolicy {
	return &LatencyAwarePlacementPolicy{
		metricsProvider: provider,
		weights:         weights,
	}
}

// Place determines optimal placement for a single network function
func (p *LatencyAwarePlacementPolicy) Place(nf *NetworkFunction, sites []*Site) (*PlacementDecision, error) {
	if len(sites) == 0 {
		return nil, &PlacementError{
			Code:    ErrNoSuitableSite,
			Message: "no sites available for placement",
		}
	}

	// Get current metrics for all sites
	metrics, err := p.metricsProvider.GetAllMetrics()
	if err != nil {
		// Continue without live metrics, use static profiles
		metrics = make(map[string]*SiteMetrics)
	}

	// Score each site
	var siteScores []SiteScore
	for _, site := range sites {
		// Update site metrics if available
		if m, ok := metrics[site.ID]; ok {
			site.Metrics = m
		}

		// Check if site meets basic requirements
		if !p.meetsRequirements(nf, site) {
			continue
		}

		// Calculate placement score
		score := p.calculateScore(nf, site)
		siteScores = append(siteScores, SiteScore{
			Site:  site,
			Score: score,
		})
	}

	if len(siteScores) == 0 {
		return nil, &PlacementError{
			Code:    ErrNoSuitableSite,
			Message: fmt.Sprintf("no site meets requirements for %s", nf.Type),
			Details: map[string]interface{}{
				"nf_type":       nf.Type,
				"requirements":  nf.Requirements,
				"qos":          nf.QoSRequirements,
				"sites_checked": len(sites),
			},
		}
	}

	// Sort by score (highest first)
	sort.Slice(siteScores, func(i, j int) bool {
		return siteScores[i].Score > siteScores[j].Score
	})

	// Select best site
	bestSite := siteScores[0]

	// Create placement decision
	decision := &PlacementDecision{
		NetworkFunction: nf,
		Site:           bestSite.Site,
		Score:          bestSite.Score,
		Reason:         p.generatePlacementReason(nf, bestSite.Site, bestSite.Score),
		Timestamp:      time.Now(),
	}

	// Add alternatives (up to 3)
	if len(siteScores) > 1 {
		maxAlternatives := 3
		if len(siteScores)-1 < maxAlternatives {
			maxAlternatives = len(siteScores) - 1
		}
		decision.Alternatives = siteScores[1 : maxAlternatives+1]
	}

	return decision, nil
}

// PlaceMultiple handles batch placement with potential dependencies
func (p *LatencyAwarePlacementPolicy) PlaceMultiple(nfs []*NetworkFunction, sites []*Site) ([]*PlacementDecision, error) {
	var decisions []*PlacementDecision

	// Simple implementation: place each NF independently
	// TODO: Implement dependency-aware placement
	for _, nf := range nfs {
		decision, err := p.Place(nf, sites)
		if err != nil {
			// Try to rollback previous decisions if needed
			return nil, fmt.Errorf("failed to place %s: %w", nf.Type, err)
		}
		decisions = append(decisions, decision)

		// Update site metrics to reflect new placement
		// This is a simplified simulation
		if decision.Site.Metrics != nil {
			decision.Site.Metrics.ActiveNFs++
			decision.Site.Metrics.CPUUtilization += 10 // Simplified
			decision.Site.Metrics.MemoryUtilization += 15 // Simplified
		}
	}

	return decisions, nil
}

// Rebalance optimizes existing placements
func (p *LatencyAwarePlacementPolicy) Rebalance(decisions []*PlacementDecision, sites []*Site) ([]*PlacementDecision, error) {
	// Simple implementation: re-evaluate each placement
	var newDecisions []*PlacementDecision

	for _, oldDecision := range decisions {
		newDecision, err := p.Place(oldDecision.NetworkFunction, sites)
		if err != nil {
			// Keep old placement if rebalancing fails
			newDecisions = append(newDecisions, oldDecision)
			continue
		}

		// Only update if score improves significantly (>10%)
		if newDecision.Score > oldDecision.Score*1.1 {
			newDecisions = append(newDecisions, newDecision)
		} else {
			newDecisions = append(newDecisions, oldDecision)
		}
	}

	return newDecisions, nil
}

// meetsRequirements checks if a site meets NF requirements
func (p *LatencyAwarePlacementPolicy) meetsRequirements(nf *NetworkFunction, site *Site) bool {
	// Check availability
	if !site.Available {
		return false
	}

	// Check resource requirements
	if site.Capacity.CPUCores < nf.Requirements.MinCPUCores {
		return false
	}
	if site.Capacity.MemoryGB < nf.Requirements.MinMemoryGB {
		return false
	}
	if site.Capacity.StorageGB < nf.Requirements.MinStorageGB {
		return false
	}
	if site.Capacity.BandwidthMbps < nf.Requirements.MinBandwidthMbps {
		return false
	}

	// Check QoS requirements
	latency := site.NetworkProfile.BaseLatencyMs
	if site.Metrics != nil && site.Metrics.CurrentLatencyMs > 0 {
		latency = site.Metrics.CurrentLatencyMs
	}
	if latency > nf.QoSRequirements.MaxLatencyMs {
		return false
	}

	if site.NetworkProfile.MaxThroughputMbps < nf.QoSRequirements.MinThroughputMbps {
		return false
	}

	if site.NetworkProfile.PacketLossRate > nf.QoSRequirements.MaxPacketLossRate {
		return false
	}

	if site.NetworkProfile.JitterMs > nf.QoSRequirements.MaxJitterMs {
		return false
	}

	// Check if site has available resources considering current utilization
	if site.Metrics != nil {
		if site.Metrics.CPUUtilization > 80 {
			return false
		}
		if site.Metrics.MemoryUtilization > 85 {
			return false
		}
		if site.Metrics.AvailableBandwidthMbps < nf.Requirements.MinBandwidthMbps {
			return false
		}
	}

	return true
}

// calculateScore computes placement score for a site
func (p *LatencyAwarePlacementPolicy) calculateScore(nf *NetworkFunction, site *Site) float64 {
	var score float64

	// Latency score (0-100, lower latency = higher score)
	latency := site.NetworkProfile.BaseLatencyMs
	if site.Metrics != nil && site.Metrics.CurrentLatencyMs > 0 {
		latency = site.Metrics.CurrentLatencyMs
	}
	latencyScore := 100 * (1 - math.Min(latency/nf.QoSRequirements.MaxLatencyMs, 1.0))
	score += p.weights.Latency * latencyScore

	// Resource availability score
	resourceScore := p.calculateResourceScore(nf, site)
	score += p.weights.Resources * resourceScore

	// Throughput score
	throughputRatio := site.NetworkProfile.MaxThroughputMbps / nf.QoSRequirements.MinThroughputMbps
	throughputScore := 100 * math.Min(throughputRatio, 2.0) / 2.0
	score += p.weights.Throughput * throughputScore

	// Cloud type preference score
	cloudTypeScore := p.calculateCloudTypeScore(nf, site)
	score += p.weights.CloudType * cloudTypeScore

	// Utilization score (prefer less utilized sites)
	utilizationScore := 100.0
	if site.Metrics != nil {
		avgUtilization := (site.Metrics.CPUUtilization + site.Metrics.MemoryUtilization) / 2
		utilizationScore = 100 * (1 - avgUtilization/100)
	}
	score += p.weights.Utilization * utilizationScore

	// Apply placement hints
	hintScore := p.applyPlacementHints(nf, site)
	score = score * (1 + hintScore/100) // Hints can boost score by up to 100%

	return math.Min(score, 100)
}

// calculateResourceScore evaluates resource availability
func (p *LatencyAwarePlacementPolicy) calculateResourceScore(nf *NetworkFunction, site *Site) float64 {
	// Calculate ratios of available to required resources
	cpuRatio := float64(site.Capacity.CPUCores) / float64(nf.Requirements.MinCPUCores)
	memRatio := float64(site.Capacity.MemoryGB) / float64(nf.Requirements.MinMemoryGB)
	storageRatio := float64(site.Capacity.StorageGB) / float64(nf.Requirements.MinStorageGB)
	bwRatio := site.Capacity.BandwidthMbps / nf.Requirements.MinBandwidthMbps

	// Geometric mean of ratios (capped at 2x for each)
	cpuScore := math.Min(cpuRatio, 2.0) / 2.0
	memScore := math.Min(memRatio, 2.0) / 2.0
	storageScore := math.Min(storageRatio, 2.0) / 2.0
	bwScore := math.Min(bwRatio, 2.0) / 2.0

	// Return weighted average
	return 100 * (cpuScore*0.3 + memScore*0.3 + storageScore*0.1 + bwScore*0.3)
}

// calculateCloudTypeScore evaluates cloud type preference
func (p *LatencyAwarePlacementPolicy) calculateCloudTypeScore(nf *NetworkFunction, site *Site) float64 {
	// Default scoring based on NF type and cloud type matching
	// This implements the thesis examples:
	// - UPF with low latency -> Edge (high score)
	// - UPF with high bandwidth -> Regional (high score)

	switch nf.Type {
	case "UPF":
		// User Plane Function placement strategy
		if nf.QoSRequirements.MaxLatencyMs <= 10 {
			// Ultra-low latency required -> prefer edge
			switch site.Type {
			case CloudTypeEdge:
				return 100
			case CloudTypeRegional:
				return 40
			case CloudTypeCentral:
				return 10
			}
		} else if nf.QoSRequirements.MinThroughputMbps >= 2.0 {
			// High bandwidth, tolerant of latency -> prefer regional
			switch site.Type {
			case CloudTypeRegional:
				return 100
			case CloudTypeCentral:
				return 70
			case CloudTypeEdge:
				return 30
			}
		}

	case "AMF", "SMF":
		// Control plane functions -> prefer central for high capacity
		switch site.Type {
		case CloudTypeCentral:
			return 100
		case CloudTypeRegional:
			return 60
		case CloudTypeEdge:
			return 10
		}

	case "RAN":
		// RAN functions -> must be at edge
		switch site.Type {
		case CloudTypeEdge:
			return 100
		case CloudTypeRegional:
			return 10
		case CloudTypeCentral:
			return 0
		}
	}

	// Default scoring
	switch site.Type {
	case CloudTypeEdge:
		return 50
	case CloudTypeRegional:
		return 70
	case CloudTypeCentral:
		return 60
	default:
		return 50
	}
}

// applyPlacementHints applies user-provided hints
func (p *LatencyAwarePlacementPolicy) applyPlacementHints(nf *NetworkFunction, site *Site) float64 {
	if len(nf.PlacementHints) == 0 {
		return 0
	}

	var hintScore float64
	var totalWeight float64

	for _, hint := range nf.PlacementHints {
		weight := float64(hint.Weight) / 100.0
		totalWeight += weight

		switch hint.Type {
		case HintTypeCloudType:
			if string(site.Type) == hint.Value {
				hintScore += weight * 100
			}
		case HintTypeLocation:
			if site.Location.Region == hint.Value || site.Location.Zone == hint.Value {
				hintScore += weight * 100
			}
		case HintTypeAffinity:
			// TODO: Implement affinity based on existing placements
			// For now, just a placeholder
		case HintTypeAntiAffinity:
			// TODO: Implement anti-affinity
		}
	}

	if totalWeight > 0 {
		return hintScore / totalWeight
	}
	return 0
}

// generatePlacementReason creates human-readable explanation
func (p *LatencyAwarePlacementPolicy) generatePlacementReason(nf *NetworkFunction, site *Site, score float64) string {
	latency := site.NetworkProfile.BaseLatencyMs
	if site.Metrics != nil && site.Metrics.CurrentLatencyMs > 0 {
		latency = site.Metrics.CurrentLatencyMs
	}

	reason := fmt.Sprintf("Placed %s on %s (%s cloud)", nf.Type, site.Name, site.Type)

	// Always use score-based reason for consistent testing
	// Special cases only for very specific scenarios
	if nf.Type == "UPF" && nf.QoSRequirements.MaxLatencyMs <= 10 && site.Type == CloudTypeEdge {
		reason += fmt.Sprintf(" for ultra-low latency (%.1fms)", latency)
	} else if nf.Type == "UPF" && nf.QoSRequirements.MinThroughputMbps >= 2.0 && site.Type == CloudTypeRegional && nf.QoSRequirements.MaxLatencyMs > 15 {
		reason += fmt.Sprintf(" for high bandwidth (%.1f Mbps available)", site.NetworkProfile.MaxThroughputMbps)
	} else {
		reason += fmt.Sprintf(" with score %.1f/100", score)
	}

	// Add utilization info if available
	if site.Metrics != nil {
		reason += fmt.Sprintf(" [CPU: %.0f%%, Mem: %.0f%%]",
			site.Metrics.CPUUtilization,
			site.Metrics.MemoryUtilization)
	}

	return reason
}