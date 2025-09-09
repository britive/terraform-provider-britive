## 2.2.0

FEATURES:
* **New Resource:**  `britive_resource_manager_response_template`: Create, update, and manage resource manager response templates.(PAB-19954)
* **New Resource:**  `britive_resource_manager_resource_type`: Create, update, and manage resource manager resource type.(PAB-19956)
* **New Resource:**  `britive_resource_manager_resource_type_permission`: Create, update, and manage resource manager resource type permissions.(PAB-19957)
* **New Resource:**  `britive_resource_manager_resource_label`: Create, update, and manage resource manager resource label.(PAB-19958)
* **New Resource:**  `britive_resource_manager_resource`: Create, update, and manage resource manager resource.(PAB-19961)
* **New Resource:**  `britive_resource_manager_resource_policy`: Create, update, and manage resource manager resource policy.(PAB-19962)
* **New Resource:**  `britive_resource_manager_profile`: Create, update, and manage resource manager profile.(PAB-19963)
* **New Resource:**  `britive_resource_manager_profile_permission`: Create, update, and manage resource manager profile permission.(PAB-19965, PAB-21493)
* **New Resource:**  `britive_resource_manager_profile_policy`: Create, update, and manager profile policy.(PAB-19966)
* **New Resource:**  `britive_resource_manager_resource_broker_pools`: Create, update, and manage resorce manager broker pools.(PAB-20093)
* **New Data Source:** `britive_escalation_policy`: Retrieve information about a specific escalation policy required for configuring IM settings (PAB-21573).
* **New Data Source:** `britive_resource_manager_profile_permissions`: Retrieve the permissions available for a specific profile. (PAB-21574)

ENHANCEMENTS:
* `resource/britive_advanced_settings`: Support for IM settings (PAB-21130). Allowing to configure advance settings for RESOURCE_MANAGER_PROFILE, RESOURCE_MANAGER_PROFILE_POLICY (PAB-21572).
* `data-source/britive_connection`: Support to fetch IM settings (PAB-21130).
* `data-source/britive_all_connections`: Support to fetch IM settings (PAB-21130).
