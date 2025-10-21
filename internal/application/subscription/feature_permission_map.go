package subscription

var FeatureToPermissionMap = map[string][]string{
	"advanced_analytics": {
		"analytics:view_advanced",
		"analytics:export",
	},
	"api_access": {
		"api:access",
		"api:create_token",
	},
	"webhook_integration": {
		"webhook:create",
		"webhook:update",
		"webhook:delete",
	},
	"custom_branding": {
		"branding:customize",
		"branding:upload_logo",
	},
	"priority_support": {
		"support:priority",
	},
	"team_collaboration": {
		"team:invite",
		"team:manage",
	},
}

func GetPermissionsForFeatures(features []string) []string {
	permissionSet := make(map[string]struct{})

	for _, feature := range features {
		if permissions, exists := FeatureToPermissionMap[feature]; exists {
			for _, perm := range permissions {
				permissionSet[perm] = struct{}{}
			}
		}
	}

	permissions := make([]string, 0, len(permissionSet))
	for perm := range permissionSet {
		permissions = append(permissions, perm)
	}

	return permissions
}
