package rabbitmq

import (
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	client := NewClient("http://localhost:15672", "guest", "guest", 10*time.Second)

	if client.baseURL != "http://localhost:15672" {
		t.Errorf("Expected baseURL to be 'http://localhost:15672', got '%s'", client.baseURL)
	}

	if client.username != "guest" {
		t.Errorf("Expected username to be 'guest', got '%s'", client.username)
	}

	if client.password != "guest" {
		t.Errorf("Expected password to be 'guest', got '%s'", client.password)
	}

	if client.httpClient.Timeout != 10*time.Second {
		t.Errorf("Expected timeout to be 10s, got '%v'", client.httpClient.Timeout)
	}
}

func TestAPIError_Error(t *testing.T) {
	apiErr := &APIError{
		ErrorMsg: "not_found",
		Reason:   "Object not found",
	}

	expected := "not_found: Object not found"
	if apiErr.Error() != expected {
		t.Errorf("Expected error message to be '%s', got '%s'", expected, apiErr.Error())
	}
}

func TestQueue_IsDeadLetterQueue(t *testing.T) {
	tests := []struct {
		name     string
		queue    Queue
		expected bool
	}{
		{
			name: "DLQ with x-dead-letter-exchange",
			queue: Queue{
				Name: "test_queue",
				Arguments: map[string]interface{}{
					"x-dead-letter-exchange": "dlx",
				},
			},
			expected: true,
		},
		{
			name: "DLQ with .dlq suffix",
			queue: Queue{
				Name:      "test_queue.dlq",
				Arguments: map[string]interface{}{},
			},
			expected: true,
		},
		{
			name: "DLQ with .dead suffix",
			queue: Queue{
				Name:      "test_queue.dead",
				Arguments: map[string]interface{}{},
			},
			expected: true,
		},
		{
			name: "DLQ with .deadletter suffix",
			queue: Queue{
				Name:      "test_queue.deadletter",
				Arguments: map[string]interface{}{},
			},
			expected: true,
		},
		{
			name: "Regular queue",
			queue: Queue{
				Name:      "test_queue",
				Arguments: map[string]interface{}{},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.queue.IsDeadLetterQueue()
			if result != tt.expected {
				t.Errorf("Expected IsDeadLetterQueue() to be %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestQueue_GetQueueState(t *testing.T) {
	tests := []struct {
		name     string
		queue    Queue
		expected QueueState
	}{
		{
			name: "Idle queue - no consumers, no messages",
			queue: Queue{
				Name:      "idle_queue",
				Consumers: 0,
				Messages:  0,
			},
			expected: QueueStateIdle,
		},
		{
			name: "Active queue - has consumers and messages",
			queue: Queue{
				Name:      "active_queue",
				Consumers: 2,
				Messages:  10,
				MessageStats: &MessageStats{
					DeliverDetails: &RateDetails{Rate: 5.0},
				},
			},
			expected: QueueStateActive,
		},
		{
			name: "Blocked queue - has consumers but no delivery rate",
			queue: Queue{
				Name:      "blocked_queue",
				Consumers: 1,
				Messages:  15,
				MessageStats: &MessageStats{
					DeliverDetails: &RateDetails{Rate: 0.0},
					PublishDetails: &RateDetails{Rate: 5.0}, // High publish rate
				},
			},
			expected: QueueStateBlocked,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.queue.GetQueueState()
			if result != tt.expected {
				t.Errorf("Expected GetQueueState() to be %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestQueue_GetPublishRate(t *testing.T) {
	queue := Queue{
		Name: "test_queue",
		MessageStats: &MessageStats{
			PublishDetails: &RateDetails{Rate: 10.5},
		},
	}

	expected := 10.5
	if queue.GetPublishRate() != expected {
		t.Errorf("Expected GetPublishRate() to be %f, got %f", expected, queue.GetPublishRate())
	}
}

func TestQueue_GetDeliverRate(t *testing.T) {
	queue := Queue{
		Name: "test_queue",
		MessageStats: &MessageStats{
			DeliverDetails: &RateDetails{Rate: 8.2},
		},
	}

	expected := 8.2
	if queue.GetDeliverRate() != expected {
		t.Errorf("Expected GetDeliverRate() to be %f, got %f", expected, queue.GetDeliverRate())
	}
}

func TestQueue_GetAckRate(t *testing.T) {
	queue := Queue{
		Name: "test_queue",
		MessageStats: &MessageStats{
			AckDetails: &RateDetails{Rate: 7.8},
		},
	}

	expected := 7.8
	if queue.GetAckRate() != expected {
		t.Errorf("Expected GetAckRate() to be %f, got %f", expected, queue.GetAckRate())
	}
}
