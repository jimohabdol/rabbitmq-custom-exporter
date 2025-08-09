package rabbitmq

import (
	"encoding/json"
	"time"
)

type Queue struct {
	Name                   string                 `json:"name"`
	Vhost                  string                 `json:"vhost"`
	Messages               int64                  `json:"messages"`
	MessagesReady          int64                  `json:"messages_ready"`
	MessagesUnacknowledged int64                  `json:"messages_unacknowledged"`
	Consumers              int64                  `json:"consumers"`
	ConsumerUtilisation    float64                `json:"consumer_utilisation"`
	MessageStats           *MessageStats          `json:"message_stats,omitempty"`
	Arguments              map[string]interface{} `json:"arguments"`
	State                  string                 `json:"state,omitempty"`
	IdleSince              *time.Time             `json:"idle_since,omitempty"`
}

type MessageStats struct {
	Publish          int64        `json:"publish"`
	PublishDetails   *RateDetails `json:"publish_details,omitempty"`
	Deliver          int64        `json:"deliver"`
	DeliverDetails   *RateDetails `json:"deliver_details,omitempty"`
	Ack              int64        `json:"ack"`
	AckDetails       *RateDetails `json:"ack_details,omitempty"`
	Redeliver        int64        `json:"redeliver"`
	RedeliverDetails *RateDetails `json:"redeliver_details,omitempty"`
}

type RateDetails struct {
	Rate float64 `json:"rate"`
}

type QueueState string

const (
	QueueStateIdle    QueueState = "idle"
	QueueStateActive  QueueState = "active"
	QueueStateBlocked QueueState = "blocked"
)

func (q *Queue) IsDeadLetterQueue() bool {
	if _, hasDLX := q.Arguments["x-dead-letter-exchange"]; hasDLX {
		return true
	}

	name := q.Name
	if len(name) >= 4 && name[len(name)-4:] == ".dlq" {
		return true
	}
	if len(name) >= 5 && name[len(name)-5:] == ".dead" {
		return true
	}
	if len(name) >= 11 && name[len(name)-11:] == ".deadletter" {
		return true
	}
	return false
}

func (q *Queue) GetQueueState() QueueState {
	if q.Consumers == 0 {
		if q.Messages == 0 {
			return QueueStateIdle
		}
		if q.MessageStats != nil && q.MessageStats.PublishDetails != nil {
			if q.MessageStats.PublishDetails.Rate < 0.01 {
				return QueueStateIdle
			}
		}
	}

	if q.Consumers > 0 && q.Messages > 0 {
		if q.MessageStats != nil && q.MessageStats.DeliverDetails != nil {
			deliveryRate := q.MessageStats.DeliverDetails.Rate
			publishRate := 0.0
			if q.MessageStats.PublishDetails != nil {
				publishRate = q.MessageStats.PublishDetails.Rate
			}

			if deliveryRate < 0.1 && publishRate > 1.0 {
				return QueueStateBlocked
			}
		}
	}

	return QueueStateActive
}

func (q *Queue) GetPublishRate() float64 {
	if q.MessageStats != nil && q.MessageStats.PublishDetails != nil {
		return q.MessageStats.PublishDetails.Rate
	}
	return 0.0
}

func (q *Queue) GetDeliverRate() float64 {
	if q.MessageStats != nil && q.MessageStats.DeliverDetails != nil {
		return q.MessageStats.DeliverDetails.Rate
	}
	return 0.0
}

func (q *Queue) GetAckRate() float64 {
	if q.MessageStats != nil && q.MessageStats.AckDetails != nil {
		return q.MessageStats.AckDetails.Rate
	}
	return 0.0
}

func (q *Queue) GetRedeliverRate() float64 {
	if q.MessageStats != nil && q.MessageStats.RedeliverDetails != nil {
		return q.MessageStats.RedeliverDetails.Rate
	}
	return 0.0
}

func (q *Queue) GetTotalRedeliveries() int64 {
	if q.MessageStats != nil {
		return q.MessageStats.Redeliver
	}
	return 0
}

type APIError struct {
	ErrorMsg string `json:"error"`
	Reason   string `json:"reason"`
}

func (e *APIError) Error() string {
	return e.ErrorMsg + ": " + e.Reason
}

func (e *APIError) UnmarshalJSON(data []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	if errorVal, ok := raw["error"].(string); ok {
		e.ErrorMsg = errorVal
	}
	if reasonVal, ok := raw["reason"].(string); ok {
		e.Reason = reasonVal
	}

	return nil
}
