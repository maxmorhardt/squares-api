package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	contestsCreatedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "contests_created_total",
			Help: "Total number of contests created",
		},
	)

	contestsDeletedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "contests_deleted_total",
			Help: "Total number of contests deleted",
		},
	)

	contestsStartedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "contests_started_total",
			Help: "Total number of contests transitioned out of the active/setup phase",
		},
	)

	quarterResultsRecordedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "quarter_results_recorded_total",
			Help: "Total number of quarter results recorded by quarter",
		},
		[]string{"quarter"},
	)

	chatMessagesTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "chat_messages_total",
			Help: "Total number of chat messages sent",
		},
	)

	invitesCreatedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "invites_created_total",
			Help: "Total number of contest invites created",
		},
	)

	invitesRedeemedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "invites_redeemed_total",
			Help: "Total number of contest invites successfully redeemed",
		},
	)

	participantsJoinedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "participants_joined_total",
			Help: "Total number of participants joined a contest by role",
		},
		[]string{"role"},
	)

	participantsRemovedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "participants_removed_total",
			Help: "Total number of participants removed from a contest",
		},
	)

	squaresClaimedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "squares_claimed_total",
			Help: "Total number of squares claimed by users",
		},
	)

	squaresClearedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "squares_cleared_total",
			Help: "Total number of squares cleared",
		},
	)
)

func init() {
	prometheus.MustRegister(
		contestsCreatedTotal,
		contestsDeletedTotal,
		contestsStartedTotal,
		quarterResultsRecordedTotal,
		chatMessagesTotal,
		invitesCreatedTotal,
		invitesRedeemedTotal,
		participantsJoinedTotal,
		participantsRemovedTotal,
		squaresClaimedTotal,
		squaresClearedTotal,
	)
}

func IncContestCreated() { 
	contestsCreatedTotal.Inc() 
}

func IncContestDeleted() { 
	contestsDeletedTotal.Inc() }

func IncContestStarted() { 
	contestsStartedTotal.Inc() 
}

func IncQuarterResult(quarter int) {
	quarterResultsRecordedTotal.WithLabelValues(quarterLabel(quarter)).Inc()
}

func IncChatMessage() { 
	chatMessagesTotal.Inc() 
}

func IncInviteCreated() { 
	invitesCreatedTotal.Inc() 
}

func IncInviteRedeemed() { 
	invitesRedeemedTotal.Inc() 
}

func IncParticipantJoined(role string) {
	participantsJoinedTotal.WithLabelValues(role).Inc()
}

func IncParticipantRemoved() { 
	participantsRemovedTotal.Inc() 
}

func IncSquareClaimed() { 
	squaresClaimedTotal.Inc() 
}

func IncSquareCleared() { 
	squaresClearedTotal.Inc() 
}

func quarterLabel(q int) string {
	switch q {
	case 1:
		return "1"
	case 2:
		return "2"
	case 3:
		return "3"
	case 4:
		return "4"
	default:
		return "unknown"
	}
}
