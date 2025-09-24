package placement

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

// OptimizedPolicy implements high-performance placement with caching and pre-computation
type OptimizedPolicy struct {
	basePolicy      *LatencyAwarePolicy
	metricsProvider MetricsProvider

	// Caching and pre-computation
	siteScoreCache    map[string]*CachedSiteScore
	placementCache    map[string]*CachedDecision
	precomputedScores map[string]map[string]float64 // [siteID][nfType] -> score

	// Performance optimization
	cacheMutex      sync.RWMutex
	lastCacheUpdate time.Time
	cacheExpiry     time.Duration

	// Statistics
	stats *Stats
}

// CachedSiteScore stores pre-computed site scoring information
type CachedSiteScore struct {
	SiteID         string
	BaseScore      float64
	ResourceRatio  float64
	NetworkScore   float64
	CloudTypePrefs map[string]float64 // [nfType] -> preference score
	LastUpdated    time.Time
	ValidUntil     time.Time
}

// CachedDecision stores recent placement decisions for reuse
type CachedDecision struct {
	Decision    *Decision
	RequestHash string
	CreatedAt   time.Time
	HitCount    int
}

// Stats tracks performance metrics
type Stats struct {
	TotalRequests     int64
	CacheHits         int64
	CacheMisses       int64
	PrecomputeHits    int64
	AvgDecisionTimeMs float64
	TotalDecisionTime time.Duration
	mu                sync.Mutex
}

// NewOptimizedPolicy creates a new optimized placement policy
func NewOptimizedPolicy(provider MetricsProvider) *OptimizedPolicy {
	return &OptimizedPolicy{
		basePolicy:        NewLatencyAwarePolicy(provider),
		metricsProvider:   provider,
		siteScoreCache:    make(map[string]*CachedSiteScore),
		placementCache:    make(map[string]*CachedDecision),
		precomputedScores: make(map[string]map[string]float64),
		cacheExpiry:       5 * time.Minute, // Cache expires after 5 minutes
		stats:             &Stats{},
	}
}

// PrecomputeSiteScores pre-computes scoring information for all sites
func (p *OptimizedPolicy) PrecomputeSiteScores(_ context.Context, sites []*Site) error {
	start := time.Now()

	// Get current metrics once for all sites
	metrics, err := p.metricsProvider.GetAllMetrics()
	if err != nil {
		// Continue with static profiles
		metrics = make(map[string]*SiteMetrics)
	}

	p.cacheMutex.Lock()
	defer p.cacheMutex.Unlock()

	// Pre-compute base scores for each site
	for _, site := range sites {
		// Update site metrics if available
		if m, ok := metrics[site.ID]; ok {
			site.Metrics = m
		}

		cachedScore := &CachedSiteScore{
			SiteID:         site.ID,
			BaseScore:      p.calculateBaseScore(site),
			ResourceRatio:  p.calculateResourceAvailabilityRatio(site),
			NetworkScore:   p.calculateNetworkScore(site),
			CloudTypePrefs: make(map[string]float64),
			LastUpdated:    time.Now(),
			ValidUntil:     time.Now().Add(p.cacheExpiry),
		}

		// Pre-compute cloud type preferences for common NF types
		nfTypes := []string{"UPF", "AMF", "SMF", "RAN", "CN"}
		for _, nfType := range nfTypes {
			cachedScore.CloudTypePrefs[nfType] = p.calculateCloudTypePreference(nfType, site)
		}

		p.siteScoreCache[site.ID] = cachedScore

		// Initialize precomputed scores for this site
		if _, exists := p.precomputedScores[site.ID]; !exists {
			p.precomputedScores[site.ID] = make(map[string]float64)
		}
	}

	p.lastCacheUpdate = time.Now()

	precomputeTime := time.Since(start)
	fmt.Printf("Pre-computed scores for %d sites in %v\n", len(sites), precomputeTime)

	return nil
}

// Place determines optimal placement using cached computations
func (p *OptimizedPolicy) Place(nf *NetworkFunction, sites []*Site) (*Decision, error) {
	start := time.Now()
	defer func() {
		p.updateStats(time.Since(start))
	}()

	if len(sites) == 0 {
		return nil, &Error{
			Code:    ErrNoSuitableSite,
			Message: "no sites available for placement",
		}
	}

	// Check placement cache first
	requestHash := p.generateRequestHash(nf, sites)
	if cached := p.getCachedDecision(requestHash); cached != nil {
		p.stats.mu.Lock()
		p.stats.CacheHits++
		p.stats.mu.Unlock()
		return cached.Decision, nil
	}

	p.stats.mu.Lock()
	p.stats.CacheMisses++
	p.stats.mu.Unlock()

	// Check if cache needs refresh
	if p.needsCacheRefresh() {
		go p.refreshCache(sites) // Async refresh
	}

	// Use fast scoring with cached values
	var siteScores []SiteScore
	for _, site := range sites {
		// Check basic requirements first (fast path)
		if !p.fastRequirementsCheck(nf, site) {
			continue
		}

		// Calculate score using cached values
		score := p.calculateOptimizedScore(nf, site)
		siteScores = append(siteScores, SiteScore{
			Site:  site,
			Score: score,
		})
	}

	if len(siteScores) == 0 {
		return nil, &Error{
			Code:    ErrNoSuitableSite,
			Message: fmt.Sprintf("no site meets requirements for %s", nf.Type),
			Details: map[string]interface{}{
				"nf_type":       nf.Type,
				"requirements":  nf.Requirements,
				"qos":           nf.QoSRequirements,
				"sites_checked": len(sites),
			},
		}
	}

	// Sort by score (highest first) - optimized sort
	if len(siteScores) > 1 {
		sort.Slice(siteScores, func(i, j int) bool {
			return siteScores[i].Score > siteScores[j].Score
		})
	}

	bestSite := siteScores[0]

	// Create placement decision
	decision := &Decision{
		NetworkFunction: nf,
		Site:            bestSite.Site,
		Score:           bestSite.Score,
		Reason:          p.generateOptimizedReason(nf, bestSite.Site, bestSite.Score),
		Timestamp:       time.Now(),
	}

	// Add alternatives (up to 3)
	if len(siteScores) > 1 {
		maxAlternatives := 3
		if len(siteScores)-1 < maxAlternatives {
			maxAlternatives = len(siteScores) - 1
		}
		decision.Alternatives = siteScores[1 : maxAlternatives+1]
	}

	// Cache the decision
	p.cacheDecision(requestHash, decision)

	return decision, nil
}

// PlaceMultipleBatch optimized batch placement with parallel processing
func (p *OptimizedPolicy) PlaceMultipleBatch(nfs []*NetworkFunction, sites []*Site) ([]*Decision, error) {
	if len(nfs) == 0 {
		return []*Decision{}, nil
	}

	// Pre-compute site scores once for all placements
	if err := p.PrecomputeSiteScores(context.Background(), sites); err != nil {
		return nil, fmt.Errorf("failed to precompute site scores: %w", err)
	}

	decisions := make([]*Decision, len(nfs))
	errors := make([]error, len(nfs))

	// Process placements in parallel using worker pool
	const maxWorkers = 4
	sem := make(chan struct{}, maxWorkers)
	var wg sync.WaitGroup

	for i, nf := range nfs {
		wg.Add(1)
		go func(index int, networkFunc *NetworkFunction) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire semaphore
			defer func() { <-sem }() // Release semaphore

			decision, err := p.Place(networkFunc, sites)
			decisions[index] = decision
			errors[index] = err

			// Update site metrics simulation for subsequent placements
			if err == nil && decision.Site.Metrics != nil {
				p.updateSiteMetricsSimulation(decision.Site, networkFunc)
			}
		}(i, nf)
	}

	wg.Wait()

	// Check for any errors
	for i, err := range errors {
		if err != nil {
			return nil, fmt.Errorf("failed to place %s: %w", nfs[i].Type, err)
		}
	}

	return decisions, nil
}

// Fast requirements check using cached metrics
func (p *OptimizedPolicy) fastRequirementsCheck(nf *NetworkFunction, site *Site) bool {
	if !site.Available {
		return false
	}

	// Check cached resource ratio
	if !p.checkCachedResources(site) {
		return false
	}

	// Check resource requirements
	if !p.checkResourceRequirements(nf, site) {
		return false
	}

	// Check QoS requirements
	if !p.checkQoSRequirements(nf, site) {
		return false
	}

	// Check utilization levels
	if !p.checkUtilizationLevels(site) {
		return false
	}

	return true
}

// checkCachedResources validates cached resource availability
func (p *OptimizedPolicy) checkCachedResources(site *Site) bool {
	p.cacheMutex.RLock()
	cached, exists := p.siteScoreCache[site.ID]
	p.cacheMutex.RUnlock()

	if exists && time.Now().Before(cached.ValidUntil) {
		// Quick resource ratio check
		if cached.ResourceRatio < 0.2 { // Less than 20% resources available
			return false
		}
	}
	return true
}

// checkResourceRequirements validates basic resource requirements
func (p *OptimizedPolicy) checkResourceRequirements(nf *NetworkFunction, site *Site) bool {
	return site.Capacity.CPUCores >= nf.Requirements.MinCPUCores &&
		site.Capacity.MemoryGB >= nf.Requirements.MinMemoryGB &&
		site.Capacity.StorageGB >= nf.Requirements.MinStorageGB &&
		site.Capacity.BandwidthMbps >= nf.Requirements.MinBandwidthMbps
}

// checkQoSRequirements validates QoS requirements
func (p *OptimizedPolicy) checkQoSRequirements(nf *NetworkFunction, site *Site) bool {
	latency := site.NetworkProfile.BaseLatencyMs
	if site.Metrics != nil && site.Metrics.CurrentLatencyMs > 0 {
		latency = site.Metrics.CurrentLatencyMs
	}

	return latency <= nf.QoSRequirements.MaxLatencyMs &&
		site.NetworkProfile.MaxThroughputMbps >= nf.QoSRequirements.MinThroughputMbps &&
		site.NetworkProfile.PacketLossRate <= nf.QoSRequirements.MaxPacketLossRate &&
		site.NetworkProfile.JitterMs <= nf.QoSRequirements.MaxJitterMs
}

// checkUtilizationLevels validates current utilization is acceptable
func (p *OptimizedPolicy) checkUtilizationLevels(site *Site) bool {
	if site.Metrics == nil {
		return true
	}
	return site.Metrics.CPUUtilization <= 80 && site.Metrics.MemoryUtilization <= 85
}

// Calculate optimized score using cached values
func (p *OptimizedPolicy) calculateOptimizedScore(nf *NetworkFunction, site *Site) float64 {
	p.cacheMutex.RLock()
	cached, exists := p.siteScoreCache[site.ID]
	p.cacheMutex.RUnlock()

	// Check if we have a pre-computed score
	if exists && time.Now().Before(cached.ValidUntil) {
		p.stats.mu.Lock()
		p.stats.PrecomputeHits++
		p.stats.mu.Unlock()

		// Use cached cloud type preference
		cloudTypeScore := cached.CloudTypePrefs[nf.Type]
		if cloudTypeScore == 0 {
			cloudTypeScore = 50.0 // Default score
		}

		// Combine cached scores with current QoS requirements
		latencyScore := p.calculateLatencyScore(nf, site)
		resourceScore := cached.BaseScore * 0.6 // Cached base resource score
		networkScore := cached.NetworkScore

		// Weighted combination
		weights := p.basePolicy.weights
		score := weights.Latency*latencyScore +
			weights.Resources*resourceScore +
			weights.CloudType*cloudTypeScore +
			weights.Throughput*networkScore

		return math.Min(score, 100.0)
	}

	// Fallback to standard calculation
	return p.basePolicy.calculateScore(nf, site)
}

// Helper methods for caching and optimization

func (p *OptimizedPolicy) calculateBaseScore(site *Site) float64 {
	// Base score based on resource availability
	if site.Metrics == nil {
		return 75.0 // Default for sites without metrics
	}

	cpuAvailable := 100.0 - site.Metrics.CPUUtilization
	memAvailable := 100.0 - site.Metrics.MemoryUtilization

	return (cpuAvailable + memAvailable) / 2.0
}

func (p *OptimizedPolicy) calculateResourceAvailabilityRatio(site *Site) float64 {
	if site.Metrics == nil {
		return 1.0
	}

	// Calculate overall resource availability ratio
	cpuRatio := (100.0 - site.Metrics.CPUUtilization) / 100.0
	memRatio := (100.0 - site.Metrics.MemoryUtilization) / 100.0

	return (cpuRatio + memRatio) / 2.0
}

func (p *OptimizedPolicy) calculateNetworkScore(site *Site) float64 {
	// Network capability score
	maxThroughput := site.NetworkProfile.MaxThroughputMbps
	baseLatency := site.NetworkProfile.BaseLatencyMs

	throughputScore := math.Min(maxThroughput/100.0, 1.0) * 50.0 // Max 50 points
	latencyScore := math.Max(0, 50.0-(baseLatency/10.0)*10.0)    // Max 50 points

	return throughputScore + latencyScore
}

func (p *OptimizedPolicy) calculateCloudTypePreference(nfType string, site *Site) float64 {
	// Use the same logic as base policy but cache the result
	return p.basePolicy.calculateCloudTypeScore(&NetworkFunction{Type: nfType}, site)
}

func (p *OptimizedPolicy) calculateLatencyScore(nf *NetworkFunction, site *Site) float64 {
	latency := site.NetworkProfile.BaseLatencyMs
	if site.Metrics != nil && site.Metrics.CurrentLatencyMs > 0 {
		latency = site.Metrics.CurrentLatencyMs
	}

	return 100 * (1 - math.Min(latency/nf.QoSRequirements.MaxLatencyMs, 1.0))
}

func (p *OptimizedPolicy) generateRequestHash(nf *NetworkFunction, sites []*Site) string {
	// Simple hash based on NF requirements and available sites
	hash := fmt.Sprintf("%s_%f_%f_%d_%d",
		nf.Type,
		nf.QoSRequirements.MaxLatencyMs,
		nf.QoSRequirements.MinThroughputMbps,
		nf.Requirements.MinCPUCores,
		len(sites))
	return hash
}

func (p *OptimizedPolicy) getCachedDecision(requestHash string) *CachedDecision {
	p.cacheMutex.RLock()
	defer p.cacheMutex.RUnlock()

	cached, exists := p.placementCache[requestHash]
	if !exists {
		return nil
	}

	// Check if cache entry is still valid (5 minutes)
	if time.Since(cached.CreatedAt) > p.cacheExpiry {
		delete(p.placementCache, requestHash)
		return nil
	}

	cached.HitCount++
	return cached
}

func (p *OptimizedPolicy) cacheDecision(requestHash string, decision *Decision) {
	p.cacheMutex.Lock()
	defer p.cacheMutex.Unlock()

	p.placementCache[requestHash] = &CachedDecision{
		Decision:    decision,
		RequestHash: requestHash,
		CreatedAt:   time.Now(),
		HitCount:    0,
	}

	// Limit cache size (LRU eviction)
	if len(p.placementCache) > 1000 {
		// Remove oldest entries
		oldestTime := time.Now()
		oldestKey := ""
		for key, cached := range p.placementCache {
			if cached.CreatedAt.Before(oldestTime) {
				oldestTime = cached.CreatedAt
				oldestKey = key
			}
		}
		if oldestKey != "" {
			delete(p.placementCache, oldestKey)
		}
	}
}

func (p *OptimizedPolicy) needsCacheRefresh() bool {
	return time.Since(p.lastCacheUpdate) > p.cacheExpiry/2
}

func (p *OptimizedPolicy) refreshCache(sites []*Site) {
	if err := p.PrecomputeSiteScores(context.Background(), sites); err != nil {
		// Log error but continue with stale cache
		// In production, this would be logged properly
		_ = err
	}
}

func (p *OptimizedPolicy) updateSiteMetricsSimulation(site *Site, _ *NetworkFunction) {
	// Simulate resource usage update for subsequent placements
	if site.Metrics != nil {
		site.Metrics.ActiveNFs++
		site.Metrics.CPUUtilization += 5    // Simplified simulation
		site.Metrics.MemoryUtilization += 8 // Simplified simulation
	}
}

func (p *OptimizedPolicy) generateOptimizedReason(nf *NetworkFunction, site *Site, score float64) string {
	// Optimized reason generation
	return fmt.Sprintf("Placed %s on %s (score: %.1f/100)", nf.Type, site.Name, score)
}

func (p *OptimizedPolicy) updateStats(duration time.Duration) {
	p.stats.mu.Lock()
	defer p.stats.mu.Unlock()

	p.stats.TotalRequests++
	p.stats.TotalDecisionTime += duration
	p.stats.AvgDecisionTimeMs = float64(p.stats.TotalDecisionTime.Nanoseconds()) / float64(p.stats.TotalRequests) / 1e6
}

// GetStats returns performance statistics
func (p *OptimizedPolicy) GetStats() *Stats {
	p.stats.mu.Lock()
	defer p.stats.mu.Unlock()

	// Return a copy to avoid race conditions
	return &Stats{
		TotalRequests:     p.stats.TotalRequests,
		CacheHits:         p.stats.CacheHits,
		CacheMisses:       p.stats.CacheMisses,
		PrecomputeHits:    p.stats.PrecomputeHits,
		AvgDecisionTimeMs: p.stats.AvgDecisionTimeMs,
		TotalDecisionTime: p.stats.TotalDecisionTime,
	}
}

// ClearCache clears all cached data
func (p *OptimizedPolicy) ClearCache() {
	p.cacheMutex.Lock()
	defer p.cacheMutex.Unlock()

	p.siteScoreCache = make(map[string]*CachedSiteScore)
	p.placementCache = make(map[string]*CachedDecision)
	p.precomputedScores = make(map[string]map[string]float64)
	p.lastCacheUpdate = time.Time{}
}
