package britive

// GetApplicationRootEnvironmentGroup - Returns root environment group
func (c *Client) GetApplicationRootEnvironmentGroup(appContainerID string) (*ApplicationRootEnvironmentGroup, error) {
	application, err := c.GetApplication(appContainerID)
	if err != nil {
		return nil, err
	}
	if application.RootEnvironmentGroup == nil {
		return &ApplicationRootEnvironmentGroup{}, nil
	}
	return application.RootEnvironmentGroup, nil
}
