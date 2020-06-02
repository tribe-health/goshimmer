package shutdown

const (
	PriorityDatabase = iota
	PriorityFPC
	PriorityTangle
	PriorityWaspConn
	PriorityMissingMessagesMonitoring
	PriorityRemoteLog
	PriorityAnalysis
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
