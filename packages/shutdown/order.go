package shutdown

const (
	PriorityDatabase = iota
	PriorityFPC
	PriorityTangle
	PriorityWaspConn
	PriorityMissingMessagesMonitoring
	PriorityFaucet
	PriorityRemoteLog
	PriorityAnalysis
	PriorityPrometheus
	PriorityMetrics
	PriorityAutopeering
	PriorityGossip
	PriorityWebAPI
	PriorityDashboard
	PrioritySynchronization
	PrioritySpammer
	PriorityBootstrap
)
