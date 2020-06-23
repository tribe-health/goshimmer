package shutdown

const (
	PriorityDatabase = iota
	PriorityFPC
	PriorityTangle
	PriorityWaspConn
	PriorityMissingMessagesMonitoring
	PriorityRemoteLog
	PriorityAnalysis
	PriorityPrometheus
	PriorityMetrics
	PriorityAutopeering
	PriorityGossip
	PriorityWebAPI
	PriorityDashboard
	PrioritySynchronization
	PriorityBootstrap
	PrioritySpammer
	PriorityBadgerGarbageCollection
)
