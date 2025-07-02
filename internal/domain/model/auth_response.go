package model

// AuthResponse is the authentication server response
type AuthResponse struct {
	Code    int    `json:"code"`
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    AuthData `json:"data"`
	Meta    AuthMeta `json:"meta"`
}

// AuthData is the structure for user data in token validation response
type AuthData struct {
	UserID       string       `json:"user_id"`
	Fullname     string       `json:"fullname"`
	Username     string       `json:"username"`
	Email        string       `json:"email"`
	Subscription Subscription `json:"subscription"`
}

// AuthMeta is the structure for metadata in token validation response
type AuthMeta struct {
	HeaderStatusCode int `json:"header_status_code"`
}

// Subscription contains user subscription information
type Subscription struct {
	Name     string            `json:"name"`
	Limits   SubscriptionLimits `json:"limits"`
	Features SubscriptionFeatures `json:"features"`
}

// SubscriptionLimits is the structure for subscription limitations
type SubscriptionLimits struct {
	Tunnels    ResourceLimit `json:"tunnels"`
	Ports      ResourceLimit `json:"ports"`
	Bandwidth  ResourceLimit `json:"bandwidth"`
	Requests   ResourceLimit `json:"requests"`
}

// ResourceLimit is the structure for resource limitations
type ResourceLimit struct {
	Limit   int  `json:"limit"`
	Used    int  `json:"used"`
	Reached bool `json:"reached"`
}

// SubscriptionFeatures is the structure for subscription features
type SubscriptionFeatures struct {
	CustomDomains    bool `json:"customDomains"`
	Analytics        bool `json:"analytics"`
	PrioritySupport  bool `json:"prioritySupport"`
}
