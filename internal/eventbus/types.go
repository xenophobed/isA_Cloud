package eventbus

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go/jetstream"
)

// Event represents a domain event
type Event struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Source    string                 `json:"source"`
	Subject   string                 `json:"subject,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
	Metadata  map[string]string      `json:"metadata,omitempty"`
	Version   string                 `json:"version,omitempty"`
}

// Command represents a command message
type Command struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Source   string                 `json:"source"`
	Target   string                 `json:"target"`
	Payload  map[string]interface{} `json:"payload"`
	Metadata map[string]string      `json:"metadata,omitempty"`
}

// CommandResult represents the result of a command execution
type CommandResult struct {
	Success bool                   `json:"success"`
	Data    map[string]interface{} `json:"data,omitempty"`
	Error   string                 `json:"error,omitempty"`
}

// EventHandler is a function that handles events
type EventHandler func(ctx context.Context, event Event) error

// CommandHandler is a function that handles commands
type CommandHandler func(ctx context.Context, command Command) (*CommandResult, error)

// ConsumerConfig configuration for event consumers
type ConsumerConfig struct {
	Durable       string
	FilterSubject string
	MaxDeliver    int
	AckWait       time.Duration
	AckPolicy     jetstream.AckPolicy
}

// ConsumerOption is a function that modifies consumer configuration
type ConsumerOption func(*ConsumerConfig)

// Common Event Types
const (
	// User Events
	EventUserCreated        = "user.created"
	EventUserUpdated        = "user.updated"
	EventUserDeleted        = "user.deleted"
	EventUserLoggedIn       = "user.logged_in"
	EventUserLoggedOut      = "user.logged_out"
	EventUserPasswordReset  = "user.password_reset"
	
	// Organization Events
	EventOrgCreated         = "organization.created"
	EventOrgUpdated         = "organization.updated"
	EventOrgDeleted         = "organization.deleted"
	EventOrgMemberAdded     = "organization.member_added"
	EventOrgMemberRemoved   = "organization.member_removed"
	
	// Payment Events
	EventPaymentInitiated   = "payment.initiated"
	EventPaymentCompleted   = "payment.completed"
	EventPaymentFailed      = "payment.failed"
	EventPaymentRefunded    = "payment.refunded"
	EventSubscriptionCreated = "subscription.created"
	EventSubscriptionCanceled = "subscription.canceled"
	
	// Task Events
	EventTaskCreated        = "task.created"
	EventTaskUpdated        = "task.updated"
	EventTaskCompleted      = "task.completed"
	EventTaskDeleted        = "task.deleted"
	EventTaskAssigned       = "task.assigned"
	
	// Notification Events
	EventNotificationSent   = "notification.sent"
	EventNotificationRead   = "notification.read"
	EventNotificationFailed = "notification.failed"
	
	// Audit Events
	EventAuditLogCreated    = "audit.log_created"
	EventSecurityAlert      = "audit.security_alert"
)

// Common Command Types
const (
	// User Commands
	CommandCreateUser       = "create_user"
	CommandUpdateUser       = "update_user"
	CommandDeleteUser       = "delete_user"
	CommandResetPassword    = "reset_password"
	
	// Organization Commands
	CommandCreateOrg        = "create_organization"
	CommandUpdateOrg        = "update_organization"
	CommandDeleteOrg        = "delete_organization"
	CommandAddMember        = "add_member"
	CommandRemoveMember     = "remove_member"
	
	// Payment Commands
	CommandProcessPayment   = "process_payment"
	CommandRefundPayment    = "refund_payment"
	CommandCreateSubscription = "create_subscription"
	CommandCancelSubscription = "cancel_subscription"
	
	// Task Commands
	CommandCreateTask       = "create_task"
	CommandUpdateTask       = "update_task"
	CommandCompleteTask     = "complete_task"
	CommandAssignTask       = "assign_task"
	
	// Notification Commands
	CommandSendNotification = "send_notification"
	CommandSendEmail        = "send_email"
	CommandSendSMS          = "send_sms"
)

// Event Sources (Services)
const (
	SourceAuthService       = "auth_service"
	SourceUserService       = "user_service"
	SourceOrgService        = "organization_service"
	SourcePaymentService    = "payment_service"
	SourceTaskService       = "task_service"
	SourceNotificationService = "notification_service"
	SourceAuditService      = "audit_service"
	SourceGateway           = "api_gateway"
)

// WithDurable sets the durable name for a consumer
func WithDurable(name string) ConsumerOption {
	return func(c *ConsumerConfig) {
		c.Durable = name
	}
}

// WithMaxDeliver sets the maximum delivery attempts
func WithMaxDeliver(max int) ConsumerOption {
	return func(c *ConsumerConfig) {
		c.MaxDeliver = max
	}
}

// WithAckWait sets the acknowledgment wait time
func WithAckWait(duration time.Duration) ConsumerOption {
	return func(c *ConsumerConfig) {
		c.AckWait = duration
	}
}

// generateEventID generates a unique event ID
func generateEventID() string {
	return uuid.New().String()
}