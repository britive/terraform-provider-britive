package britive

//Tag - godoc
type Tag struct {
	ID                       string                    `json:"userTagId,omitempty"`
	Name                     string                    `json:"name"`
	Description              string                    `json:"description"`
	Status                   string                    `json:"status,omitempty"`
	UserCount                interface{}               `json:"userCount,omitempty"`
	UserTagIdentityProviders []UserTagIdentityProvider `json:"userTagIdentityProviders,omitempty"`
	External                 interface{}               `json:"external,omitempty"`
}

//UserTagIdentityProvider - godoc
type UserTagIdentityProvider struct {
	IdentityProvider IdentityProvider `json:"identityProvider"`
	ExternalID       interface{}      `json:"externalId,omitempty"`
}

//IdentityProvider - godoc
type IdentityProvider struct {
	ID          string `json:"id"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type,omitempty"`
}

//User - godoc
type User struct {
	AdminRoles       []AdminRole      `json:"adminRoles,omitempty"`
	Type             string           `json:"type,omitempty"`
	Email            string           `json:"email,omitempty"`
	Username         string           `json:"username,omitempty"`
	FirstName        string           `json:"firstName,omitempty"`
	LastName         string           `json:"lastName,omitempty"`
	Name             string           `json:"name,omitempty"`
	ExternalID       interface{}      `json:"externalId,omitempty"`
	Mobile           interface{}      `json:"mobile,omitempty"`
	IdentityProvider IdentityProvider `json:"identityProvider,omitempty"`
	MappedAccounts   []interface{}    `json:"mappedAccounts,omitempty"`
	External         bool             `json:"external,omitempty"`
	Status           string           `json:"status,omitempty"`
	UserID           string           `json:"userId,omitempty"`
}

//AdminRole - godoc
type AdminRole struct {
	Name        string `json:"name,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
}

//Profile - godoc
type Profile struct {
	ProfileID                     string      `json:"papId,omitempty"`
	AppContainerID                string      `json:"appContainerId"`
	Scopes                        []Scope     `json:"scope"`
	Name                          string      `json:"name"`
	Description                   string      `json:"description"`
	Status                        string      `json:"status,omitempty"`
	ExpirationDuration            int64       `json:"expirationDuration,omitempty"`
	Extendable                    bool        `json:"extendable"`
	NotificationPriorToExpiration *int64      `json:"notificationPriorToExpiration,omitempty"`
	ExtensionDuration             *int64      `json:"extensionDuration,omitempty"`
	ExtensionLimit                interface{} `json:"extensionLimit,omitempty"`
}

//Scope - godoc
type Scope struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

//Application - godoc
type Application struct {
	AppContainerID        string `json:"appContainerId"`
	CatalogAppDisplayName string `json:"catalogAppDisplayName,omitempty"`
}
