package events

import "time"

type WAFEvent struct {
	Timestamp       time.Time
	RequestID       string
	TenantID        string
	AppID           string
	PolicyID        string
	PolicyVersionID string
	Host            string
	ClientIP        string
	Method          string
	Path            string
	Action          string
	Mode            string
	Status          uint16
	Reason          string
	MatchedRuleID   string
	MatchedRuleName string
	RuleGroup       string
	Tags            []string
	AnomalyScore    int32
	UserAgent       string
	LatencyMS       uint32
	OriginStatus    uint16
	OriginLatencyMS uint32
}

// TODO: Add worker jobs for ClickHouse retention cleanup, enrichment, and
// backfills once the async job runner is implemented.
